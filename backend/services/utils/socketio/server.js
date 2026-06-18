const dotenv = require('dotenv')
dotenv.config();
dotenv.config({ path: `.env.local`, override: true });
const express = require('express');
const jwt = require('jsonwebtoken');
const app = express();
const PORT = 5000;

const http = require('http').Server(app);
const cors = require('cors');
var allowedOrigins;
let allowedOriginsFromEnv = (process.env.CORS_ALLOWED_ORIGINS || '').split(',').filter(Boolean)
if (allowedOriginsFromEnv.length > 1) {
  allowedOrigins = allowedOriginsFromEnv
} else if (allowedOriginsFromEnv.length === 1) {
  allowedOrigins = allowedOriginsFromEnv[0]
} else {
  allowedOrigins = "*"
}
console.log("allowedOrigins:", allowedOrigins)

const JWT_SECRET = process.env.SECRET_API_KEY || "supersecretkey"

const io = require('socket.io')(http, {
    cors: {
        origin: allowedOrigins
    }
});

app.use(cors());

// -------- JWT auth middleware for Socket.IO --------
// Clients must pass token in handshake auth: { token: "Bearer <jwt>" }
io.use((socket, next) => {
    const token = socket.handshake.auth && socket.handshake.auth.token;
    if (!token) {
        return next(new Error('Authentication required: no token provided'));
    }
    const raw = token.startsWith('Bearer ') ? token.slice(7) : token;
    try {
        const claims = jwt.verify(raw, JWT_SECRET, { algorithms: ['HS256'] });
        socket.data.email = claims.email || claims.Username || '';
        next();
    } catch (err) {
        console.warn(`[Auth] Rejected connection: ${err.message}`);
        next(new Error('Authentication failed: invalid or expired token'));
    }
});
// ---------------------------------------------------

let users = []

// -------- NATS → Socket.IO bridge for USP Notify --------
const NATS_URL = process.env.NATS_URL || "nats://nats:4222"
const WEBHOOK_URL = process.env.WEBHOOK_URL || ""

// natsConn is set once the NATS bridge starts; used for device-existence validation.
let natsConn = null;

async function startNatsBridgeWithRef() {
    try {
        const { connect, StringCodec } = require('nats');
        const sc = StringCodec();
        const nc = await connect({ servers: NATS_URL });
        natsConn = nc;
        console.log(`[NATS] Connected to ${NATS_URL}`);

        const sub = nc.subscribe("notification.v1.>");
        console.log("[NATS] Subscribed to notification.v1.>");

        (async () => {
            for await (const msg of sub) {
                try {
                    const data = sc.decode(msg.data);
                    const notification = JSON.parse(data);
                    console.log(`[NATS] USP Notify from ${notification.device_sn}`);

                    // Deliver to all authenticated users (tenant:all) + device-specific rooms.
                    // tenant:all ensures every logged-in user receives notifications without
                    // needing to call subscribe_device. Device rooms allow device-specific
                    // subscriptions for future granular filtering.
                    const deviceRoom = `device:${notification.device_sn}`;
                    io.to('tenant:all').to(deviceRoom).emit('usp_notify', notification);

                    if (WEBHOOK_URL) {
                        fetch(WEBHOOK_URL, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: data,
                        }).catch(err => console.error('[Webhook] Push failed:', err));
                    }
                } catch (e) {
                    console.error('[NATS] Failed to process notification:', e);
                }
            }
        })();

        (async () => {
            for await (const s of nc.status()) {
                console.info(`[NATS] Status: ${s.type}`);
            }
        })();

    } catch (err) {
        console.error('[NATS] Connection failed:', err);
        setTimeout(startNatsBridgeWithRef, 5000);
    }
}

startNatsBridgeWithRef();

io.on('connection', (socket) => {
    const email = socket.data.email;
    console.log(`🚀: ${socket.id} (${email}) connected`);

    // All authenticated users join the shared tenant room so they receive all
    // device notifications without needing to call subscribe_device explicitly.
    socket.join('tenant:all');

    // Admin role also joins the legacy admin room for compatibility.
    const isAdmin = (process.env.ADMIN_EMAILS || '').split(',').map(s => s.trim()).includes(email);
    if (isAdmin) {
        socket.join('admin');
    }

    // subscribe_device: validate device existence via NATS before joining the room.
    // This prevents subscribing to non-existent or unauthorised device names.
    // Full RBAC (per S7) will replace this placeholder with a tenant-ownership check.
    socket.on('subscribe_device', async (deviceSN) => {
        if (typeof deviceSN !== 'string' || !deviceSN.trim()) return;
        const sn = deviceSN.trim();

        if (!natsConn) {
            socket.emit('subscribe_error', { device_sn: sn, reason: 'Service unavailable' });
            return;
        }

        try {
            const msg = await natsConn.request(`adapter.usp.v1.${sn}.device`, new Uint8Array(), { timeout: 3000 });
            const resp = JSON.parse(new TextDecoder().decode(msg.data));
            if (resp.Code === 200) {
                socket.join(`device:${sn}`);
                console.log(`[Sub] ${email} subscribed to device:${sn}`);
                socket.emit('subscribe_ok', { device_sn: sn });
            } else {
                console.warn(`[Sub] ${email} denied for device:${sn} (code=${resp.Code})`);
                socket.emit('subscribe_error', { device_sn: sn, reason: 'Device not found or access denied' });
            }
        } catch (err) {
            console.warn(`[Sub] device validation failed for ${sn}: ${err.message}`);
            socket.emit('subscribe_error', { device_sn: sn, reason: 'Device not found or access denied' });
        }
    });

    socket.on('unsubscribe_device', (deviceSN) => {
        if (typeof deviceSN === 'string' && deviceSN.trim()) {
            socket.leave(`device:${deviceSN.trim()}`);
        }
    });

    socket.on("callUser", ({ userToCall, signalData, from }) => {
        let index = users.findIndex(x => x.name === userToCall)
        if (index >= 0) {
            io.to(users[index].id).emit("callUser", { signal: signalData, from });
        } else {
            console.log("No user named " + userToCall + " found or offline")
        }
    });

    socket.on("answerCall", (data) => {
        io.to(data.to).emit("callAccepted", data.signal);
    });

    socket.on("newuser", (data) => {
        let index = users.findIndex(x => x.name === data.name)
        if (index < 0) {
            users.push(data)
        }
        io.emit('users', users)
    })

    socket.on('disconnect', () => {
        console.log(`🔥: ${socket.id} (${email}) disconnected`);
        let index = users.findIndex(x => x.id === socket.id)
        if (index >= 0) {
            users.splice(index, 1)
            io.emit('users', users)
        }
    });
});

http.listen(PORT, () => {
    console.log(`Server listening on ${PORT}`);
});

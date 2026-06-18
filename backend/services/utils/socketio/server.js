const dotenv = require('dotenv')
dotenv.config();
dotenv.config({ path: `.env.local`, override: true });
const express = require('express');
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

const io = require('socket.io')(http, {
    cors: {
        origin: allowedOrigins
    }
});

app.use(cors());

let users = []

// -------- NATS → Socket.IO bridge for USP Notify --------
const NATS_URL = process.env.NATS_URL || "nats://nats:4222"
const WEBHOOK_URL = process.env.WEBHOOK_URL || ""

async function startNatsBridge() {
    try {
        const { connect, StringCodec } = require('nats');
        const sc = StringCodec();
        const nc = await connect({ servers: NATS_URL });
        console.log(`[NATS] Connected to ${NATS_URL}`);

        const sub = nc.subscribe("notification.v1.>");
        console.log("[NATS] Subscribed to notification.v1.>");

        (async () => {
            for await (const msg of sub) {
                try {
                    const data = sc.decode(msg.data);
                    const notification = JSON.parse(data);
                    console.log(`[NATS] USP Notify from ${notification.device_sn}`);
                    io.emit('usp_notify', notification);

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
        setTimeout(startNatsBridge, 5000);
    }
}

startNatsBridge();
// --------------------------------------------------------

io.on('connection', (socket) => {
    console.log(`🚀: ${socket.id} user just connected!`);

    socket.on("callUser", ({ userToCall, signalData, from }) => {
        console.log("user to call:", userToCall)
        let index = users.findIndex(x => x.name === userToCall)
        if (index >= 0) {
            io.to(users[index].id).emit("callUser", { signal: signalData, from });
        } else {
            console.log("There is no user named " + userToCall + " or he/she is offline")
        }
    });

    socket.on("answerCall", (data) => {
        io.to(data.to).emit("callAccepted", data.signal);
    });

    socket.on("newuser", (data) => {
        let index = users.findIndex(x => x.name === data.name)
        if (index >= 0) {
            console.log("user already exists, but got connected with other id")
        } else {
            users.push(data)
        }
        console.log(data)
        console.log("total users: ", users)
        io.emit('users', users)
    })

    socket.on('disconnect', () => {
        console.log('🔥: A user disconnected');
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

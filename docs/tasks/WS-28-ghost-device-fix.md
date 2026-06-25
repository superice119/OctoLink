# WS-28: Ghost Device / Empty-SN Fix & WS-24 RBAC Regression

## Root Cause Analysis (code-path, no runtime)

### Confirmed: Two independent defects

**Defect 1 — Ghost record (empty SN) in MongoDB**
- **Path**: USP/CWMP MTP adapter → `HandleDeviceInfo` → `parseDeviceInfoMsg` → `CreateDevice`
- **Trigger**: Any malformed USP GetResp (failed proto-unmarshal, nil body, unsupported record type, fewer than 5 `ReqPathResults`) causes `parseDeviceInfoMsg` to return early with `db.Device{}` (SN="")
- **Effect**: `CreateDevice` upserts a document with `sn: ""` into MongoDB. This row appears in the device list with all fields blank and status Offline.
- **Confirmed blocker for super_admin**: `api/device.go` lines 27-33 returns 400 "No id provided" when `?id=` is empty, before any RBAC check — the super_admin delete button silently fails.

**Defect 2 — WS-24 RBAC regression (empty customer)**
- **Path**: `api/device.go` DELETE handler, lines 39-50
- **Trigger**: When a non-super_admin (e.g. tenant_admin) tries to delete a device with `customer == ""` (unassigned), the check `device.Customer != callerTenantID` evaluates `"" != "some-tenant-id"` → true → 403 Forbidden
- **Effect**: Legitimate tenant_admin cannot delete any device that was never assigned to a tenant (e.g. device registered before WS-24 RBAC was merged)
- **Note**: This does NOT affect super_admin (the entire RBAC block is behind `if callerRole != db.RoleSuperAdmin`). The boss's symptom is Defect 1; Defect 2 is a regression that affects tenant admins.

---

## Changes

### 1. `backend/services/mtp/adapter/internal/events/usp_handler/info.go`
Guard added after `parseDeviceInfoMsg`: if `deviceInfo.SN == ""`, log and return — prevents ghost record creation AND suppresses the spurious `device.v1.new` NATS publish that would otherwise fire.

### 2. `backend/services/mtp/adapter/internal/events/cwmp_handler/info.go`
Same guard applied to the CWMP path.

### 3. `backend/services/mtp/adapter/internal/db/device.go`
Belt-and-suspenders: `CreateDevice` returns an error immediately if `device.SN == ""`. Prevents any future callers from writing ghost records.

### 4. `backend/services/controller/internal/api/device.go`
Two changes:
- **Ghost cleanup**: When `?id=` is empty AND `?cleanup_ghosts=true` AND caller is super_admin, delete all documents where `sn: ""` (explicit query param prevents accidental bulk delete from a missing id bug)
- **RBAC fix**: Changed `if device.Customer != callerTenantID` → `if device.Customer != "" && device.Customer != callerTenantID`. Devices with no customer (unassigned) may be deleted by any authorized role, consistent with `assignDeviceTenant` which already allows claiming unassigned devices.

### 5. `frontend/src/pages/devices.js`
Two changes:
- **Ghost device UX**: Delete confirmation dialog detects empty SN and shows a specific message. The Apply button is disabled for non-super_admin (since ghost cleanup requires super_admin). When SN is empty and user is super_admin, the delete call hits `?cleanup_ghosts=true`.
- **Count refresh**: Replaced local `devices.splice()` (which left `total` stale) with a `fetchDevicePerPage` re-fetch after any successful delete — satisfies "列表与计数正确刷新".

---

## Assumptions & QA Checkpoints

- **Ghost record storage format**: Fix assumes ghost records are stored as `sn: ""` (empty string) in MongoDB. If the field is absent (`sn: null` or missing), the cleanup filter `{sn: ""}` will not match. QA should `db.devices.find({sn: {$in: ["", null]}})` to confirm storage format on the live instance (`39.97.250.156`) and adjust if needed.
- **Super_admin role**: The boss is assumed to be a super_admin. Validate via `db.users.findOne({email: "..."})` if needed.
- **No Mongo migration script**: Cleanup is done through the API (`DELETE /api/device?cleanup_ghosts=true`) to respect the Awaiting Approval 铁律 for direct DB mutations. QA triggers cleanup via curl or the UI ghost-device dialog.

---

## Acceptance Checklist

- [ ] Device list no longer shows empty-SN rows after fix is deployed (prevention path)
- [ ] Existing ghost records can be deleted by super_admin via the dialog
- [ ] Tenant_admin can delete unassigned (customer="") devices with valid SN
- [ ] After delete, total count and list both refresh correctly
- [ ] QA Playwright test (WS-29 sibling) can reproduce ghost row via mocked malformed USP message, confirm it is rejected, and verify cleanup flow

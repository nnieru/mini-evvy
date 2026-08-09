# Mini Evvy — Ops User Manual

**Booking seats & scanning attendance QR**

This guide is for **ops users** (organization role: `member`). Ops can manage guests, create bookings, record payments, and check guests in at the door. Event setup (categories, seats, finalize seating) is done by staff (`owner` / `admin`).

---

## 1. Before you start

You need:

1. A Mini Evvy account with access to the event’s organization.
2. Your login email and password.
3. Categories and seats already set up for the event (staff).
4. For camera check-in: a phone or laptop on **HTTPS** or **localhost**, and permission to use the camera.

### Sign in

1. Open the Mini Evvy app.
2. Go to **Login**.
3. Enter your email and password.
4. After login, open your organization / **My events**, then open the event.

### Where ops work

| Menu | What you do there |
|------|-------------------|
| **Guests** | View guests; add a guest manually |
| **Bookings** | Select seats, create bookings, mark paid, resend invites |
| **Check-in** | Scan ticket QR / barcode, or check in by guest + seat |

Staff-only pages (Overview, Categories, Seats, Jobs, Invite email) are not available to ops.

---

## 2. Booking seats

**Page:** Event → **Bookings**

Use this when a guest pays or reserves on site, or when you need to assign seats manually.

### 2.1 Seat map rules

- Only **available** seats can be selected.
- Selected seats must be in the **same category** (same color / category on the legend).
- If you see **“Seating locked for review”**, staff is reviewing finalize seating. Wait until they approve or reject before creating new bookings.
- Unpaid holds that stay `pending` / unpaid too long may be cancelled automatically (about **1 hour**) and the seat freed.

### 2.2 Create a booking

1. Open **Bookings**.
2. On the **Seat map**, click one or more available seats in the same category.  
   - Use the category legend to see colors.  
   - **Clear** removes your selection.
3. Click **Continue**.
4. In **Guest details**:
   - Default: one **Guest name** and **Guest email** for all selected seats.
   - Or turn on **Different guest per seat** and fill name/email per seat.
   - Optional: **Notes**.
5. Click **Create N booking(s)**.
6. Success message: `Created N booking(s)`.

What happens:

- Seats become **reserved**.
- Each booking gets a barcode like `EVVY-` + 12 hex characters.
- New bookings start as **unpaid** (status may show **Pending invite** until payment is settled).

### 2.3 Manage bookings (table)

Below the seat map, the **Bookings** table lists active bookings (cancelled ones are hidden).

**Filters:** All · Paid · Unpaid

| Column | Meaning |
|--------|---------|
| Seat | Seat code |
| Guest | Name and email |
| Payment | **Paid** or **Unpaid** |
| Barcode | Ticket code used for QR / check-in |
| Actions | Buttons below |

#### Mark paid / Mark unpaid

- **Mark paid** — booking is paid; seat becomes **occupied**. Guest can be checked in.
- **Mark unpaid** — reverts a paid booking; seat goes back to **reserved**. Check-in will fail until paid again.

#### Record payment (Add payment)

Use when you want to log amount / method (not only toggle paid):

1. Open **Add payment** (or equivalent payment action) on an unpaid booking.
2. Enter **Amount** (e.g. `150000.00`), **Currency** (default `IDR`), **Status**, optional **Method**.
3. Set status to **Success** when payment is confirmed.
4. Click **Save payment**.

A **Success** payment marks the booking paid (when eligible) and typically queues the invitation email with the QR ticket.

#### Resend invite

1. Click **Resend invite** on the booking.
2. Wait for success: invitation resent in the background.
3. Guest must have an email; cancelled bookings cannot be resent.

#### Delete booking

Deletes (cancels) the booking and frees the seat. It disappears from the active list.

### 2.4 Typical ops booking path

```
Select seats → Create booking(s) → Mark paid (or Save payment Success)
→ Guest receives email with QR / ticket code → Ready for Check-in
```

**Important:** Check-in only works for **paid** bookings.

---

## 3. Scanning attendance QR (check-in)

**Page:** Event → **Check-in**

Description in app: *Scan tickets with your phone or laptop camera, or enter a barcode manually.*

### 3.1 Prerequisites

- Booking is **Paid**.
- Guest has a barcode (shown in Bookings; also on the invitation email as QR + ticket code).
- For camera mode: allow camera access; use HTTPS or localhost.

### 3.2 Mode A — Camera (recommended at the door)

1. Open **Check-in**.
2. Select **Camera**.
3. Pick a camera from the dropdown (prefer rear camera on phones).
4. Click **Start camera**.
5. Point the camera at the guest’s ticket **QR code**.
6. On success: **Guest checked in successfully**.  
   The scanner pauses briefly so the same code is not scanned twice in a row.
7. Click **Stop camera** when finished, or leave it running for the next guest.

Tips:

- If no cameras appear, click **Refresh cameras** and allow browser permission.
- Works with QR and common barcodes (e.g. Code 128).
- Phone or laptop both work.

### 3.3 Mode B — Manual barcode

Use when the camera fails, or with a USB barcode wedge:

1. Select **Manual barcode**.
2. Type or scan into **Barcode** (format `EVVY-...`).
3. Click **Check in**.

### 3.4 Mode C — Guest + seat (fallback)

Use when the guest has no QR / barcode but you know who they are and their seat:

1. Select **Guest + seat**.
2. Choose **Guest** and **Seat**.
3. Optional **Message**.
4. Click **Check in**.

There must still be a **paid** booking for that guest + seat pair.

### 3.5 Attendance log

The **Attendance log** lists check-ins with time and status:

- **Checked in** — guest is in; use **Undo** if you checked the wrong person.
- After **Undo**, status becomes **not checked in** and they can be checked in again.

---

## 4. Common messages & fixes

| Message / situation | What it means | What to do |
|---------------------|---------------|------------|
| Seating locked for review | Finalize seating is in preview | Ask staff to approve or reject on Event overview |
| No categories / No seats yet | Setup incomplete | Ask staff to add categories and seats |
| Cannot select seats across categories | Mixed categories selected | Clear selection; pick seats in one category only |
| Paid booking not found | Unpaid, wrong event, or invalid code | Confirm booking is **Paid** and barcode matches Bookings |
| Already checked in | Guest already in | Use **Undo** only if the check-in was a mistake |
| Camera does not start | Browser blocked camera / not secure context | Use HTTPS or localhost; allow camera; Refresh cameras |
| Resend invite failed | Missing email or cancelled booking | Fix guest email; ensure booking is active |
| Guest email missing | Invite needs an email | Edit / recreate with a valid email |

---

## 5. Quick day-of checklist

**Before doors open**

- [ ] Logged in on phone or laptop (HTTPS if not localhost)
- [ ] Open the correct event → **Check-in** → **Camera**
- [ ] Test one known paid ticket QR
- [ ] Confirm unpaid guests are marked **Paid** on **Bookings** if they should enter

**At the door**

- [ ] Keep **Camera** running
- [ ] Scan QR → wait for success → next guest
- [ ] If scan fails: check **Bookings** (Paid? correct barcode?) → try Manual barcode → or Guest + seat

**After mistakes**

- [ ] Use **Undo** on the wrong check-in in the Attendance log
- [ ] Re-scan the correct ticket

---

## 6. What ops cannot do (ask staff)

- Create / edit categories and seats
- Finalize, approve, or reject seating
- Import guests from spreadsheet
- Edit invitation email templates
- View Jobs / Event overview finalize controls

Ops **can** add individual guests on **Guests**, create and manage bookings, record payment / mark paid, resend invites, and run check-in.

---

## 7. Status cheat sheet

| Item | Values that matter for ops |
|------|----------------------------|
| Booking payment (UI) | **Paid** = can check in · **Unpaid** = cannot check in |
| Seat on map | **available** = selectable · **reserved** / **occupied** = already booked |
| Attendance | **checked_in** · **not_checked_in** (after Undo) |

---

*Product: Mini Evvy (`mini-evvy`). For API and staff setup details, see the repository `README.md` and `frontend/README.md`.*

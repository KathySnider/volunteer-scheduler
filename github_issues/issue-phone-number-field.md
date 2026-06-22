# Issue: Remove or Clarify Unused Phone Number Field (PII Reduction)

## Description
The user profile and volunteer/staff management sections currently include a **Phone Number** input field. However, there is no text messaging (SMS) or telephony integration (e.g., Twilio) in the application.

Given the beta testing environment constraints and general data privacy best practices, collecting and storing unnecessary Personally Identifiable Information (PII) like phone numbers introduces security and compliance risks without providing functional value to the scheduling system.

## Proposed Changes
Unless text messaging notifications are planned for a future phase, it is recommended to:
1. **Frontend**: Remove the Phone Number input fields from the Volunteer profile settings page, Admin Volunteer management form, and Admin Staff management form.
2. **Backend**: Remove the `phone` field from the GraphQL schemas and DB queries.
3. **Database**: Drop the `phone` columns from the `volunteers` and `staff` tables.

---

## Technical Details & Code References

### 1. Frontend Profile Settings Page
The phone number input is present in the profile component:
* **File**: `frontend/src/app/profile/page.js`
* **Line 81**: Maps GraphQL response to form state:
  ```javascript
  phone: p.phone ?? "",
  ```
* **Lines 211–224**: Renders the input control:
  ```jsx
  <div className={styles.field}>
    <label className={styles.label} htmlFor="phone">
      Phone
    </label>
    <input
      id="phone"
      className={styles.input}
      type="tel"
      value={form.phone}
      placeholder="(555) 555-5555"
      onChange={(e) => setForm((p) => ({ ...p, phone: e.target.value }))}
    />
  </div>
  ```

### 2. Frontend Admin Volunteer Management Page
* **File**: `frontend/src/app/admin/volunteers/page.js`
* **Lines 117–122**: Admin form fields for creating/editing volunteer profiles collect the phone number.
* **Line 465**: Displays the volunteer's phone number on the roster/list.

### 3. Frontend Admin Staff Management Page
* **File**: `frontend/src/app/admin/staff/page.js`
* **Lines 91–96**: Admin form fields for staff detail collection.
* **Line 352**: Displays the staff member's phone number.

### 4. Backend GraphQL Schema
* **File**: `backend/graph/volunteer/schema.graphql` (Lines 118, 169)
* **File**: `backend/graph/admin/schema.graphql` (Lines 206, 228, 404, 413, 453, 464)

### 5. Database Schema
* **File**: `backend/migrations/000001_init.up.sql`
* **Line 46**: `phone VARCHAR(20)` on `volunteers` table.
* **Line 72**: `phone VARCHAR(20)` on `staff` table.

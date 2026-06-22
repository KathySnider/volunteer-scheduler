# Issue: Pre-emptive Overlap Warnings on Non-selected Shifts Create Confusing UX

## Description
When a volunteer signs up for a shift, the UI immediately scans all other visible shifts on the page. Any other shift that overlaps with the signed-up shift is immediately styled with:
1. A yellow warning banner: `⚠️ Overlaps with your shift: [Event Name]`
2. The sign-up button label changed from `"Sign Up"` to `"Sign Up Anyway"`.

This pre-emptive warning occurs **before** the user has interacted with or expressed intent to sign up for the overlapping shift. This design causes confusion, as volunteers feel they have already done something wrong or are being flagged for the shift they *just* successfully signed up for. 

Instead of showing warning states pre-emptively on list rows, the system should only warn the user and require confirmation (e.g., "Sign Up Anyway") when they actually attempt to sign up for a conflicting shift.

## Visual Evidence
![Overlap Warning Screenshot](https://raw.githubusercontent.com/KathySnider/volunteer-scheduler/bug-reports/github_issues/conflict-warning-screenshot.png)

## Proposed Resolution
1. **Remove Pre-emptive Warning Display**: Do not show the conflict banner or transition the button to "Sign Up Anyway" on page/list load.
2. **Warn on Click/Action**: 
   - When a volunteer clicks "Sign Up" on a shift that has conflicts, intercept the action.
   - Show a confirmation modal/dialog warning them about the overlap.
   - If they select "Sign Up Anyway" in the modal, proceed with the signup request.

---

## Technical Details & Code References

### Frontend Shift Details Rendering
* **File**: `frontend/src/app/events/[id]/page.js`
* **Line 172**: The `hasConflict` boolean is calculated pre-emptively for every list row during render:
  ```javascript
  const hasConflict = !isSignedUp && conflictingShifts && conflictingShifts.length > 0;
  ```
* **Lines 192–202**: Changes the styling/label of the action button pre-emptively:
  ```jsx
  } else if (!isFull) {
    btn = (
      <button
        className={hasConflict ? styles.btnSignUpAnyway : styles.btnSignUp}
        disabled={busy}
        onClick={() => onSignUp(shift.id)}
      >
        {busy ? "Signing up…" : hasConflict ? "Sign Up Anyway" : "Sign Up"}
      </button>
    );
  }
  ```
* **Lines 205–218**: Renders the warning inline in the shift row details block:
  ```jsx
  let conflictNote = null;
  if (hasConflict) {
    const names = conflictingShifts.map((c) => c.eventName).filter(Boolean);
    const label = names.length === 1
      ? names[0]
      : names.length === 2
        ? `${names[0]} and ${names[1]}`
        : `${names[0]} and ${names.length - 1} others`;
    conflictNote = (
      <div className={styles.conflictWarning}>
        ⚠️ Overlaps with your shift: <strong>{label}</strong>
      </div>
    );
  }
  ```

# Issue: Map View Container Renders Over Filter Dropdown Panels (z-index Overlap)

## Description
When viewing the main **Volunteer Events** page in Map View, expanding any multi-select filter dropdown (e.g., the "Job" filter) results in the dropdown panel being partially obscured by the map container.

This occurs because Leaflet map components generate positioned elements (e.g., `.leaflet-pane`, `.leaflet-marker-pane`, and `.leaflet-control`) with standard library `z-index` values ranging up to `800` or higher. Because the filter bar container (`.filterBar`) is set to `z-index: 20`, its child dropdown elements (`.checkboxPanel`, `z-index: 50`) are restricted to the filter bar's low stacking context and render behind the map elements.

## Visual Evidence
![Map View z-index Overlap](https://raw.githubusercontent.com/KathySnider/volunteer-scheduler/bug-reports/github_issues/map-z-index-screenshot.png)

## Proposed Resolution
To resolve this issue, the CSS z-index layering must be restructured to ensure filter dropdowns and global site navigation menus always overlay the map content.

### Recommended Z-Index Scale:
1. **Filter Bar Container (`.filterBar`)**: Increase to `z-index: 1000` (so it and its children overlay all Leaflet panes).
2. **Top Navigation Bar (`.topBar` / `AdminTopBar`)**: Increase to `z-index: 1100` (to stay on top of the filter bar).
3. **User Profile Dropdown Menu (`.userMenu` / `.dropdown`)**: Increase to `z-index: 1200`.
4. **Feedback Button / Modal**: Increase to `z-index: 2000` (to overlay all page elements and navigation).

---

## Technical Details & Code References

### 1. Events Page Styling (Filter Bar & Dropdowns)
* **File**: `frontend/src/app/events/events.module.css`
* **Line 25**: Current top bar z-index is too low:
  ```css
  .topBar {
    ...
    z-index: 30;
  }
  ```
* **Line 83**: Current filter bar z-index is too low to overlay leaflet panes:
  ```css
  .filterBar {
    ...
    z-index: 20;
  }
  ```

### 2. Admin Top Bar Component Styling
* **File**: `frontend/src/app/components/admin-top-bar.module.css`
* **Line 11**: Stacking index needs alignment:
  ```css
  .topBar {
    ...
    z-index: 30;
  }
  ```

### 3. User Menu Dropdown Styling
* **File**: `frontend/src/app/components/UserMenu.module.css`
* **Line 65**: Needs to remain higher than the navigation header:
  ```css
  .dropdown {
    ...
    z-index: 100;
  }
  ```

### 4. Feedback Button & Modal Styling
* **File**: `frontend/src/app/components/FeedbackButton.module.css`
* **Line 11**: Trigger button:
  ```css
  .btn {
    ...
    z-index: 100;
  }
  ```
* **Line 42**: Overlay Modal:
  ```css
  .modal {
    ...
    z-index: 200;
  }
  ```

# System-test screenshots

Copy the captured PNGs into `software_design/images/system-test/` in the final
LaTeX report, using the following names.  `system-test-cases.tex` already
contains the matching figures and captions.

| File name | Captured evidence |
| --- | --- |
| `stc-01-admin-dashboard.png` | Administrator dashboard after sign-in |
| `stc-02-availability-form.png` | Availability form with a teacher selected |
| `stc-03-booking-context.png` | Booking context selected |
| `stc-04-engine-criteria.png` | Engine preferences selected |
| `stc-04-engine-no-result.png` | Engine no-results response |
| `stc-05-empty-cart.png` | Empty booking cart |
| `stc-06-teacher-management.png` | Teacher management |
| `stc-06-branch-management.png` | Branch management |
| `stc-06-commute-management.png` | Commute configuration |
| `stc-07-account-management.png` | Account management |
| `stc-08-guest-availability.png` | Guest availability browsing |
| `stc-08-student-classes.png` | Student My Classes |
| `stc-08-parent-classes.png` | Parent My Classes |

The evidence macro renders a framed fallback while a named PNG is absent, so a
draft remains compilable.  It automatically switches to the screenshot when
the file is copied into the report image directory.

# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: auth.spec.ts >> Logout >> sign-out via UI clears session and redirects to /login
- Location: tests\e2e\auth.spec.ts:148:7

# Error details

```
Test timeout of 30000ms exceeded.
```

```
Error: locator.click: Target page, context or browser has been closed
Call log:
  - waiting for getByRole('button', { name: /menu|account|settings/i }).first()

```

# Page snapshot

```yaml
- generic [active] [ref=e1]:
  - generic [ref=e2]:
    - generic [ref=e3]:
      - generic [ref=e4]: AARP Volunteer Events
      - generic [ref=e5]:
        - link "My Shifts" [ref=e6] [cursor=pointer]:
          - /url: /my-shifts
        - link "My Feedback" [ref=e7] [cursor=pointer]:
          - /url: /my-feedback
        - generic [ref=e8]:
          - generic [ref=e9]: Test Volunteer
          - link "My Profile" [ref=e10] [cursor=pointer]:
            - /url: /profile
          - button "Sign out" [ref=e11] [cursor=pointer]
    - generic [ref=e14]:
      - generic [ref=e15]:
        - generic [ref=e16]: City
        - button "All Cities" [ref=e18] [cursor=pointer]:
          - text: All Cities
          - generic [ref=e19]: ▾
      - generic [ref=e20]:
        - generic [ref=e21]: Job
        - button "All Jobs" [ref=e23] [cursor=pointer]:
          - text: All Jobs
          - generic [ref=e24]: ▾
      - generic [ref=e25]:
        - generic [ref=e26]: Format
        - combobox "Format" [ref=e27]:
          - option "All formats" [selected]
          - option "Virtual"
          - option "In Person"
          - option "Hybrid"
      - generic [ref=e28]:
        - generic [ref=e29]: Timeframe
        - combobox "Timeframe" [ref=e30]:
          - option "Upcoming" [selected]
          - option "Past"
          - option "All"
    - main [ref=e31]:
      - heading "Volunteer Events(33)" [level=1] [ref=e33]:
        - text: Volunteer Events
        - generic [ref=e34]: (33)
      - generic [ref=e35]:
        - generic [ref=e36]:
          - generic [ref=e37]:
            - generic [ref=e38]: Virtual Fraud Prevention Seminar
            - generic [ref=e39]:
              - generic [ref=e40]:
                - generic [ref=e41]: 📅
                - text: Apr 30, 2026, 6:00 AM
              - generic [ref=e42]:
                - generic [ref=e43]: 📍
                - text: Virtual
              - generic [ref=e44]: Virtual
            - paragraph [ref=e45]: Online session covering the latest scams targeting seniors and how to stay safe.
            - generic [ref=e46]:
              - generic [ref=e47]: "Roles needed:"
              - generic [ref=e48]: Event Support (1/1)
              - generic [ref=e49]: Speaker (0/3)
          - generic [ref=e50]:
            - generic [ref=e51]:
              - generic [ref=e52]: Volunteers
              - generic [ref=e53]: 1/4
              - generic [ref=e54]: Spots open
            - button "View Details" [ref=e55] [cursor=pointer]
        - generic [ref=e56]:
          - generic [ref=e57]:
            - generic [ref=e58]: Hybrid Benefits Counseling Day
            - generic [ref=e59]:
              - generic [ref=e60]:
                - generic [ref=e61]: 📅
                - text: May 3, 2026, 2:00 AM
              - generic [ref=e62]:
                - generic [ref=e63]: 📍
                - text: Tacoma, WA
              - generic [ref=e64]: Hybrid
            - paragraph [ref=e65]: One-on-one benefits counseling available both in person and via video call. Volunteers help with check-in and virtual waiting room management.
            - generic [ref=e66]:
              - generic [ref=e67]: "Roles needed:"
              - generic [ref=e68]: Event Support (0/3)
              - generic [ref=e69]: Volunteer Lead (0/2)
          - generic [ref=e70]:
            - generic [ref=e71]:
              - generic [ref=e72]: Volunteers
              - generic [ref=e73]: 0/5
              - generic [ref=e74]: Spots open
            - button "View Details" [ref=e75] [cursor=pointer]
        - generic [ref=e76]:
          - generic [ref=e77]:
            - generic [ref=e78]: Medicare Q&A Workshop
            - generic [ref=e79]:
              - generic [ref=e80]:
                - generic [ref=e81]: 📅
                - text: May 10, 2026, 2:00 AM
              - generic [ref=e82]:
                - generic [ref=e83]: 📍
                - text: Seattle, WA
              - generic [ref=e84]: In Person
            - paragraph [ref=e85]: Help seniors navigate Medicare enrollment and plan options. Volunteers assist with one-on-one sessions.
            - generic [ref=e86]:
              - generic [ref=e87]: "Roles needed:"
              - generic [ref=e88]: Advocacy (2/2)
              - generic [ref=e89]: Event Support (2/8)
          - generic [ref=e90]:
            - generic [ref=e91]:
              - generic [ref=e92]: Volunteers
              - generic [ref=e93]: 4/10
              - generic [ref=e94]: Spots open
            - button "View Details" [ref=e95] [cursor=pointer]
        - generic [ref=e96]:
          - generic [ref=e97]:
            - generic [ref=e98]: Tax Aide Preparation - Spring Session
            - generic [ref=e99]:
              - generic [ref=e100]:
                - generic [ref=e101]: 📅
                - text: May 17, 2026, 12:00 PM
              - generic [ref=e102]:
                - generic [ref=e103]: 📍
                - text: Bellevue, WA
              - generic [ref=e104]: In Person
            - paragraph [ref=e105]: Free tax preparation assistance for low-to-moderate income seniors. Training provided.
            - generic [ref=e106]:
              - generic [ref=e107]: "Roles needed:"
              - generic [ref=e108]: Event Support (1/24)
          - generic [ref=e109]:
            - generic [ref=e110]:
              - generic [ref=e111]: Volunteers
              - generic [ref=e112]: 1/24
              - generic [ref=e113]: Spots open
            - button "View Details" [ref=e114] [cursor=pointer]
        - generic [ref=e115]:
          - generic [ref=e116]:
            - generic [ref=e117]: Driver Safety Course
            - generic [ref=e118]:
              - generic [ref=e119]:
                - generic [ref=e120]: 📅
                - text: May 24, 2026, 1:30 AM
              - generic [ref=e121]:
                - generic [ref=e122]: 📍
                - text: Spokane Valley, WA
              - generic [ref=e123]: In Person
            - paragraph [ref=e124]: AARP Smart Driver course for seniors. Volunteers help with registration and materials.
            - generic [ref=e125]:
              - generic [ref=e126]: "Roles needed:"
              - generic [ref=e127]: Volunteer Lead (0/2)
          - generic [ref=e128]:
            - generic [ref=e129]:
              - generic [ref=e130]: Volunteers
              - generic [ref=e131]: 0/2
              - generic [ref=e132]: Spots open
            - button "View Details" [ref=e133] [cursor=pointer]
        - generic [ref=e134]:
          - generic [ref=e135]:
            - generic [ref=e136]: Kathy's birthday
            - generic [ref=e137]:
              - generic [ref=e138]:
                - generic [ref=e139]: 📅
                - text: Jun 6, 2026, 2:00 PM
              - generic [ref=e140]:
                - generic [ref=e141]: 📍
                - text: Virtual
              - generic [ref=e142]: Virtual
            - paragraph [ref=e143]: Happy Birthday!
            - generic [ref=e144]:
              - generic [ref=e145]: "Roles needed:"
              - generic [ref=e146]: Attendee Only (0/1)
          - generic [ref=e147]:
            - generic [ref=e148]:
              - generic [ref=e149]: Volunteers
              - generic [ref=e150]: 0/1
              - generic [ref=e151]: Spots open
            - button "View Details" [ref=e152] [cursor=pointer]
        - generic [ref=e153]:
          - generic [ref=e154]:
            - generic [ref=e155]: Spokane Senior Health Fair
            - generic [ref=e156]:
              - generic [ref=e157]:
                - generic [ref=e158]: 📅
                - text: Jun 7, 2026, 2:00 AM
              - generic [ref=e159]:
                - generic [ref=e160]: 📍
                - text: Spokane, WA
              - generic [ref=e161]: In Person
            - paragraph [ref=e162]: Community health fair with blood pressure checks, medication reviews, and wellness resources.
            - generic [ref=e163]:
              - generic [ref=e164]: "Roles needed:"
              - generic [ref=e165]: Event Support (2/10)
          - generic [ref=e166]:
            - generic [ref=e167]:
              - generic [ref=e168]: Volunteers
              - generic [ref=e169]: 2/10
              - generic [ref=e170]: Spots open
            - button "View Details" [ref=e171] [cursor=pointer]
        - generic [ref=e172]:
          - generic [ref=e173]:
            - generic [ref=e174]: Social Security Benefits Workshop
            - generic [ref=e175]:
              - generic [ref=e176]:
                - generic [ref=e177]: 📅
                - text: Jun 14, 2026, 3:00 AM
              - generic [ref=e178]:
                - generic [ref=e179]: 📍
                - text: Vancouver, WA
              - generic [ref=e180]: In Person
            - paragraph [ref=e181]: Informational session on maximizing Social Security benefits. Volunteers greet and assist attendees.
            - generic [ref=e182]:
              - generic [ref=e183]: "Roles needed:"
              - generic [ref=e184]: Event Support (0/4)
          - generic [ref=e185]:
            - generic [ref=e186]:
              - generic [ref=e187]: Volunteers
              - generic [ref=e188]: 0/4
              - generic [ref=e189]: Spots open
            - button "View Details" [ref=e190] [cursor=pointer]
        - generic [ref=e191]:
          - generic [ref=e192]:
            - generic [ref=e193]: Caregiver Support Forum
            - generic [ref=e194]:
              - generic [ref=e195]:
                - generic [ref=e196]: 📅
                - text: Jun 21, 2026, 6:00 AM
              - generic [ref=e197]:
                - generic [ref=e198]: 📍
                - text: Olympia, WA
              - generic [ref=e199]: In Person
            - paragraph [ref=e200]: Forum connecting family caregivers with local resources and support networks.
            - generic [ref=e201]:
              - generic [ref=e202]: "Roles needed:"
              - generic [ref=e203]: Event Support (1/3)
          - generic [ref=e204]:
            - generic [ref=e205]:
              - generic [ref=e206]: Volunteers
              - generic [ref=e207]: 1/3
              - generic [ref=e208]: Spots open
            - button "View Details" [ref=e209] [cursor=pointer]
        - generic [ref=e210]:
          - generic [ref=e211]:
            - generic [ref=e212]: ShiftEvent1776372093578
            - generic [ref=e213]:
              - generic [ref=e214]:
                - generic [ref=e215]: 📅
                - text: Jun 15, 2027, 6:00 AM
              - generic [ref=e216]:
                - generic [ref=e217]: 📍
                - text: Testville, VA
              - generic [ref=e218]: In Person
            - paragraph [ref=e219]: Test event
            - generic [ref=e220]:
              - generic [ref=e221]: "Roles needed:"
              - generic [ref=e222]: Greeter1776372093584 (2/5)
          - generic [ref=e223]:
            - generic [ref=e224]:
              - generic [ref=e225]: Volunteers
              - generic [ref=e226]: 2/5
              - generic [ref=e227]: Spots open
            - button "View Details" [ref=e228] [cursor=pointer]
        - generic [ref=e229]:
          - generic [ref=e230]:
            - generic [ref=e231]: ShiftEvent1776372442889
            - generic [ref=e232]:
              - generic [ref=e233]:
                - generic [ref=e234]: 📅
                - text: Jun 15, 2027, 6:00 AM
              - generic [ref=e235]:
                - generic [ref=e236]: 📍
                - text: Testville, VA
              - generic [ref=e237]: In Person
            - paragraph [ref=e238]: Test event
            - generic [ref=e239]:
              - generic [ref=e240]: "Roles needed:"
              - generic [ref=e241]: Greeter1776372442895 (2/5)
          - generic [ref=e242]:
            - generic [ref=e243]:
              - generic [ref=e244]: Volunteers
              - generic [ref=e245]: 2/5
              - generic [ref=e246]: Spots open
            - button "View Details" [ref=e247] [cursor=pointer]
        - generic [ref=e248]:
          - generic [ref=e249]:
            - generic [ref=e250]: ShiftEvent1776372533384
            - generic [ref=e251]:
              - generic [ref=e252]:
                - generic [ref=e253]: 📅
                - text: Jun 15, 2027, 6:00 AM
              - generic [ref=e254]:
                - generic [ref=e255]: 📍
                - text: Testville, VA
              - generic [ref=e256]: In Person
            - paragraph [ref=e257]: Test event
            - generic [ref=e258]:
              - generic [ref=e259]: "Roles needed:"
              - generic [ref=e260]: Greeter1776372533390 (2/5)
          - generic [ref=e261]:
            - generic [ref=e262]:
              - generic [ref=e263]: Volunteers
              - generic [ref=e264]: 2/5
              - generic [ref=e265]: Spots open
            - button "View Details" [ref=e266] [cursor=pointer]
        - generic [ref=e267]:
          - generic [ref=e268]:
            - generic [ref=e269]: FullEvent1776372093588
            - generic [ref=e270]:
              - generic [ref=e271]:
                - generic [ref=e272]: 📅
                - text: Jul 10, 2027, 5:00 AM
              - generic [ref=e273]:
                - generic [ref=e274]: 📍
                - text: Testville, VA
              - generic [ref=e275]: In Person
            - paragraph [ref=e276]: Test event
            - generic [ref=e277]:
              - generic [ref=e278]: "Roles needed:"
              - generic [ref=e279]: Checker1776372093594 (1/1)
          - generic [ref=e280]:
            - generic [ref=e281]:
              - generic [ref=e282]: Volunteers
              - generic [ref=e283]: 1/1
              - generic [ref=e284]: Fully staffed
            - button "View Details" [ref=e285] [cursor=pointer]
        - generic [ref=e286]:
          - generic [ref=e287]:
            - generic [ref=e288]: FullEvent1776372442899
            - generic [ref=e289]:
              - generic [ref=e290]:
                - generic [ref=e291]: 📅
                - text: Jul 10, 2027, 5:00 AM
              - generic [ref=e292]:
                - generic [ref=e293]: 📍
                - text: Testville, VA
              - generic [ref=e294]: In Person
            - paragraph [ref=e295]: Test event
            - generic [ref=e296]:
              - generic [ref=e297]: "Roles needed:"
              - generic [ref=e298]: Checker1776372442905 (1/1)
          - generic [ref=e299]:
            - generic [ref=e300]:
              - generic [ref=e301]: Volunteers
              - generic [ref=e302]: 1/1
              - generic [ref=e303]: Fully staffed
            - button "View Details" [ref=e304] [cursor=pointer]
        - generic [ref=e305]:
          - generic [ref=e306]:
            - generic [ref=e307]: FullEvent1776372533394
            - generic [ref=e308]:
              - generic [ref=e309]:
                - generic [ref=e310]: 📅
                - text: Jul 10, 2027, 5:00 AM
              - generic [ref=e311]:
                - generic [ref=e312]: 📍
                - text: Testville, VA
              - generic [ref=e313]: In Person
            - paragraph [ref=e314]: Test event
            - generic [ref=e315]:
              - generic [ref=e316]: "Roles needed:"
              - generic [ref=e317]: Checker1776372533400 (1/1)
          - generic [ref=e318]:
            - generic [ref=e319]:
              - generic [ref=e320]: Volunteers
              - generic [ref=e321]: 1/1
              - generic [ref=e322]: Fully staffed
            - button "View Details" [ref=e323] [cursor=pointer]
        - generic [ref=e324]:
          - generic [ref=e325]:
            - generic [ref=e326]: UpcomingEvent1776372082623
            - generic [ref=e327]:
              - generic [ref=e328]:
                - generic [ref=e329]: 📅
                - text: Aug 1, 2027, 6:00 AM
              - generic [ref=e330]:
                - generic [ref=e331]: 📍
                - text: FilterCity1776372082622, WA
              - generic [ref=e332]: In Person
            - paragraph [ref=e333]: Test event
            - generic [ref=e334]:
              - generic [ref=e335]: "Roles needed:"
              - generic [ref=e336]: Filter Role1776372082626 (0/4)
          - generic [ref=e337]:
            - generic [ref=e338]:
              - generic [ref=e339]: Volunteers
              - generic [ref=e340]: 0/4
              - generic [ref=e341]: Spots open
            - button "View Details" [ref=e342] [cursor=pointer]
        - generic [ref=e343]:
          - generic [ref=e344]:
            - generic [ref=e345]: UpcomingEvent1776372424471
            - generic [ref=e346]:
              - generic [ref=e347]:
                - generic [ref=e348]: 📅
                - text: Aug 1, 2027, 6:00 AM
              - generic [ref=e349]:
                - generic [ref=e350]: 📍
                - text: FilterCity1776372424470, WA
              - generic [ref=e351]: In Person
            - paragraph [ref=e352]: Test event
            - generic [ref=e353]:
              - generic [ref=e354]: "Roles needed:"
              - generic [ref=e355]: Filter Role1776372424474 (0/4)
          - generic [ref=e356]:
            - generic [ref=e357]:
              - generic [ref=e358]: Volunteers
              - generic [ref=e359]: 0/4
              - generic [ref=e360]: Spots open
            - button "View Details" [ref=e361] [cursor=pointer]
        - generic [ref=e362]:
          - generic [ref=e363]:
            - generic [ref=e364]: UpcomingEvent1776372533336
            - generic [ref=e365]:
              - generic [ref=e366]:
                - generic [ref=e367]: 📅
                - text: Aug 1, 2027, 6:00 AM
              - generic [ref=e368]:
                - generic [ref=e369]: 📍
                - text: FilterCity1776372533335, WA
              - generic [ref=e370]: In Person
            - paragraph [ref=e371]: Test event
            - generic [ref=e372]:
              - generic [ref=e373]: "Roles needed:"
              - generic [ref=e374]: Filter Role1776372533339 (0/4)
          - generic [ref=e375]:
            - generic [ref=e376]:
              - generic [ref=e377]: Volunteers
              - generic [ref=e378]: 0/4
              - generic [ref=e379]: Spots open
            - button "View Details" [ref=e380] [cursor=pointer]
        - generic [ref=e381]:
          - generic [ref=e382]:
            - generic [ref=e383]: AdminUpcomingEvent1776460445126
            - generic [ref=e384]:
              - generic [ref=e385]:
                - generic [ref=e386]: 📅
                - text: Aug 5, 2027, 6:00 AM
              - generic [ref=e387]:
                - generic [ref=e388]: 📍
                - text: AdminFilterCity1776460445125, WA
              - generic [ref=e389]: In Person
            - paragraph [ref=e390]: Test event
            - generic [ref=e391]:
              - generic [ref=e392]: "Roles needed:"
              - generic [ref=e393]: Admin Filter Role1776460445129 (0/5)
          - generic [ref=e394]:
            - generic [ref=e395]:
              - generic [ref=e396]: Volunteers
              - generic [ref=e397]: 0/5
              - generic [ref=e398]: Spots open
            - button "View Details" [ref=e399] [cursor=pointer]
        - generic [ref=e400]:
          - generic [ref=e401]:
            - generic [ref=e402]: AdminTFUpcoming1776460445135
            - generic [ref=e403]:
              - generic [ref=e404]:
                - generic [ref=e405]: 📅
                - text: Sep 1, 2027, 6:00 AM
              - generic [ref=e406]:
                - generic [ref=e407]: 📍
                - text: AdminTFCity1776460445134, OR
              - generic [ref=e408]: In Person
            - paragraph [ref=e409]: Test event
            - generic [ref=e410]:
              - generic [ref=e411]: "Roles needed:"
              - generic [ref=e412]: Admin TF Role1776460445138 (0/5)
          - generic [ref=e413]:
            - generic [ref=e414]:
              - generic [ref=e415]: Volunteers
              - generic [ref=e416]: 0/5
              - generic [ref=e417]: Spots open
            - button "View Details" [ref=e418] [cursor=pointer]
        - generic [ref=e419]:
          - generic [ref=e420]:
            - generic [ref=e421]: InPersonEvent1776372082638
            - generic [ref=e422]:
              - generic [ref=e423]:
                - generic [ref=e424]: 📅
                - text: Sep 15, 2027, 7:00 AM
              - generic [ref=e425]:
                - generic [ref=e426]: 📍
                - text: FormatCity1776372082637, OR
              - generic [ref=e427]: In Person
            - paragraph [ref=e428]: Test event
            - generic [ref=e429]:
              - generic [ref=e430]: "Roles needed:"
              - generic [ref=e431]: Format Role1776372082640 (0/5)
          - generic [ref=e432]:
            - generic [ref=e433]:
              - generic [ref=e434]: Volunteers
              - generic [ref=e435]: 0/5
              - generic [ref=e436]: Spots open
            - button "View Details" [ref=e437] [cursor=pointer]
        - generic [ref=e438]:
          - generic [ref=e439]:
            - generic [ref=e440]: InPersonEvent1776372424486
            - generic [ref=e441]:
              - generic [ref=e442]:
                - generic [ref=e443]: 📅
                - text: Sep 15, 2027, 7:00 AM
              - generic [ref=e444]:
                - generic [ref=e445]: 📍
                - text: FormatCity1776372424485, OR
              - generic [ref=e446]: In Person
            - paragraph [ref=e447]: Test event
            - generic [ref=e448]:
              - generic [ref=e449]: "Roles needed:"
              - generic [ref=e450]: Format Role1776372424488 (0/5)
          - generic [ref=e451]:
            - generic [ref=e452]:
              - generic [ref=e453]: Volunteers
              - generic [ref=e454]: 0/5
              - generic [ref=e455]: Spots open
            - button "View Details" [ref=e456] [cursor=pointer]
        - generic [ref=e457]:
          - generic [ref=e458]:
            - generic [ref=e459]: InPersonEvent1776372533351
            - generic [ref=e460]:
              - generic [ref=e461]:
                - generic [ref=e462]: 📅
                - text: Sep 15, 2027, 7:00 AM
              - generic [ref=e463]:
                - generic [ref=e464]: 📍
                - text: FormatCity1776372533350, OR
              - generic [ref=e465]: In Person
            - paragraph [ref=e466]: Test event
            - generic [ref=e467]:
              - generic [ref=e468]: "Roles needed:"
              - generic [ref=e469]: Format Role1776372533353 (0/5)
          - generic [ref=e470]:
            - generic [ref=e471]:
              - generic [ref=e472]: Volunteers
              - generic [ref=e473]: 0/5
              - generic [ref=e474]: Spots open
            - button "View Details" [ref=e475] [cursor=pointer]
        - generic [ref=e476]:
          - generic [ref=e477]:
            - generic [ref=e478]: AdminInPersonEvent1776460445144
            - generic [ref=e479]:
              - generic [ref=e480]:
                - generic [ref=e481]: 📅
                - text: Oct 1, 2027, 6:00 AM
              - generic [ref=e482]:
                - generic [ref=e483]: 📍
                - text: AdminFmtCity1776460445143, CA
              - generic [ref=e484]: In Person
            - paragraph [ref=e485]: Test event
            - generic [ref=e486]:
              - generic [ref=e487]: "Roles needed:"
              - generic [ref=e488]: Admin Fmt Role1776460445146 (0/5)
          - generic [ref=e489]:
            - generic [ref=e490]:
              - generic [ref=e491]: Volunteers
              - generic [ref=e492]: 0/5
              - generic [ref=e493]: Spots open
            - button "View Details" [ref=e494] [cursor=pointer]
        - generic [ref=e495]:
          - generic [ref=e496]:
            - generic [ref=e497]: ListedEvent1776372068160
            - generic [ref=e498]:
              - generic [ref=e499]:
                - generic [ref=e500]: 📅
                - text: Oct 5, 2027, 7:00 AM
              - generic [ref=e501]:
                - generic [ref=e502]: 📍
                - text: Baltimore, MD
              - generic [ref=e503]: In Person
            - paragraph [ref=e504]: Test event
            - generic [ref=e505]:
              - generic [ref=e506]: "Roles needed:"
              - generic [ref=e507]: Tabling1776372068166 (0/5)
          - generic [ref=e508]:
            - generic [ref=e509]:
              - generic [ref=e510]: Volunteers
              - generic [ref=e511]: 0/5
              - generic [ref=e512]: Spots open
            - button "View Details" [ref=e513] [cursor=pointer]
        - generic [ref=e514]:
          - generic [ref=e515]:
            - generic [ref=e516]: ListedEvent1776372424456
            - generic [ref=e517]:
              - generic [ref=e518]:
                - generic [ref=e519]: 📅
                - text: Oct 5, 2027, 7:00 AM
              - generic [ref=e520]:
                - generic [ref=e521]: 📍
                - text: Baltimore, MD
              - generic [ref=e522]: In Person
            - paragraph [ref=e523]: Test event
            - generic [ref=e524]:
              - generic [ref=e525]: "Roles needed:"
              - generic [ref=e526]: Tabling1776372424462 (0/5)
          - generic [ref=e527]:
            - generic [ref=e528]:
              - generic [ref=e529]: Volunteers
              - generic [ref=e530]: 0/5
              - generic [ref=e531]: Spots open
            - button "View Details" [ref=e532] [cursor=pointer]
        - generic [ref=e533]:
          - generic [ref=e534]:
            - generic [ref=e535]: ListedEvent1776372533321
            - generic [ref=e536]:
              - generic [ref=e537]:
                - generic [ref=e538]: 📅
                - text: Oct 5, 2027, 7:00 AM
              - generic [ref=e539]:
                - generic [ref=e540]: 📍
                - text: Baltimore, MD
              - generic [ref=e541]: In Person
            - paragraph [ref=e542]: Test event
            - generic [ref=e543]:
              - generic [ref=e544]: "Roles needed:"
              - generic [ref=e545]: Tabling1776372533327 (0/5)
          - generic [ref=e546]:
            - generic [ref=e547]:
              - generic [ref=e548]: Volunteers
              - generic [ref=e549]: 0/5
              - generic [ref=e550]: Spots open
            - button "View Details" [ref=e551] [cursor=pointer]
        - generic [ref=e552]:
          - generic [ref=e553]:
            - generic [ref=e554]: ListedEvent1776460445116
            - generic [ref=e555]:
              - generic [ref=e556]:
                - generic [ref=e557]: 📅
                - text: Oct 5, 2027, 7:00 AM
              - generic [ref=e558]:
                - generic [ref=e559]: 📍
                - text: Baltimore, MD
              - generic [ref=e560]: In Person
            - paragraph [ref=e561]: Test event
            - generic [ref=e562]:
              - generic [ref=e563]: "Roles needed:"
              - generic [ref=e564]: Tabling1776460445122 (0/5)
          - generic [ref=e565]:
            - generic [ref=e566]:
              - generic [ref=e567]: Volunteers
              - generic [ref=e568]: 0/5
              - generic [ref=e569]: Spots open
            - button "View Details" [ref=e570] [cursor=pointer]
        - generic [ref=e571]:
          - generic [ref=e572]:
            - generic [ref=e573]: CardEvent1776372082648
            - generic [ref=e574]:
              - generic [ref=e575]:
                - generic [ref=e576]: 📅
                - text: Oct 20, 2027, 6:00 AM
              - generic [ref=e577]:
                - generic [ref=e578]: 📍
                - text: CardCity1776372082647, CA
              - generic [ref=e579]: In Person
            - paragraph [ref=e580]: Test event
            - generic [ref=e581]:
              - generic [ref=e582]: "Roles needed:"
              - generic [ref=e583]: Card Role1776372082650 (0/5)
          - generic [ref=e584]:
            - generic [ref=e585]:
              - generic [ref=e586]: Volunteers
              - generic [ref=e587]: 0/5
              - generic [ref=e588]: Spots open
            - button "View Details" [ref=e589] [cursor=pointer]
        - generic [ref=e590]:
          - generic [ref=e591]:
            - generic [ref=e592]: CardEvent1776372091342
            - generic [ref=e593]:
              - generic [ref=e594]:
                - generic [ref=e595]: 📅
                - text: Oct 20, 2027, 6:00 AM
              - generic [ref=e596]:
                - generic [ref=e597]: 📍
                - text: CardCity1776372091341, CA
              - generic [ref=e598]: In Person
            - paragraph [ref=e599]: Test event
            - generic [ref=e600]:
              - generic [ref=e601]: "Roles needed:"
              - generic [ref=e602]: Card Role1776372091344 (0/5)
          - generic [ref=e603]:
            - generic [ref=e604]:
              - generic [ref=e605]: Volunteers
              - generic [ref=e606]: 0/5
              - generic [ref=e607]: Spots open
            - button "View Details" [ref=e608] [cursor=pointer]
        - generic [ref=e609]:
          - generic [ref=e610]:
            - generic [ref=e611]: CardEvent1776372424496
            - generic [ref=e612]:
              - generic [ref=e613]:
                - generic [ref=e614]: 📅
                - text: Oct 20, 2027, 6:00 AM
              - generic [ref=e615]:
                - generic [ref=e616]: 📍
                - text: CardCity1776372424495, CA
              - generic [ref=e617]: In Person
            - paragraph [ref=e618]: Test event
            - generic [ref=e619]:
              - generic [ref=e620]: "Roles needed:"
              - generic [ref=e621]: Card Role1776372424498 (0/5)
          - generic [ref=e622]:
            - generic [ref=e623]:
              - generic [ref=e624]: Volunteers
              - generic [ref=e625]: 0/5
              - generic [ref=e626]: Spots open
            - button "View Details" [ref=e627] [cursor=pointer]
        - generic [ref=e628]:
          - generic [ref=e629]:
            - generic [ref=e630]: CardEvent1776372442867
            - generic [ref=e631]:
              - generic [ref=e632]:
                - generic [ref=e633]: 📅
                - text: Oct 20, 2027, 6:00 AM
              - generic [ref=e634]:
                - generic [ref=e635]: 📍
                - text: CardCity1776372442866, CA
              - generic [ref=e636]: In Person
            - paragraph [ref=e637]: Test event
            - generic [ref=e638]:
              - generic [ref=e639]: "Roles needed:"
              - generic [ref=e640]: Card Role1776372442869 (0/5)
          - generic [ref=e641]:
            - generic [ref=e642]:
              - generic [ref=e643]: Volunteers
              - generic [ref=e644]: 0/5
              - generic [ref=e645]: Spots open
            - button "View Details" [ref=e646] [cursor=pointer]
        - generic [ref=e647]:
          - generic [ref=e648]:
            - generic [ref=e649]: CardEvent1776372533361
            - generic [ref=e650]:
              - generic [ref=e651]:
                - generic [ref=e652]: 📅
                - text: Oct 20, 2027, 6:00 AM
              - generic [ref=e653]:
                - generic [ref=e654]: 📍
                - text: CardCity1776372533360, CA
              - generic [ref=e655]: In Person
            - paragraph [ref=e656]: Test event
            - generic [ref=e657]:
              - generic [ref=e658]: "Roles needed:"
              - generic [ref=e659]: Card Role1776372533363 (0/5)
          - generic [ref=e660]:
            - generic [ref=e661]:
              - generic [ref=e662]: Volunteers
              - generic [ref=e663]: 0/5
              - generic [ref=e664]: Spots open
            - button "View Details" [ref=e665] [cursor=pointer]
    - button "Submit feedback" [ref=e666] [cursor=pointer]: "?"
  - alert [ref=e667]
```

# Test source

```ts
  57  |       page.getByRole("heading", { name: "Check your email" })
  58  |     ).toBeVisible();
  59  |     await expect(page.getByText(volunteerEmail)).toBeVisible();
  60  | 
  61  |     // 4. Fetch the email from Mailhog and extract the magic link
  62  |     const msg = await waitForEmail(volunteerEmail);
  63  |     const magicUrl = extractMagicLink(msg);
  64  | 
  65  |     // 5. Navigate to the magic-link URL
  66  |     await page.goto(magicUrl);
  67  | 
  68  |     // 6. Should show "Signed in!" then redirect to /events
  69  |     await expect(page.getByRole("heading", { name: "Signed in!" })).toBeVisible();
  70  |     await page.waitForURL("**/events", { timeout: 8_000 });
  71  |     expect(page.url()).toContain("/events");
  72  |   });
  73  | 
  74  |   test("session token is stored in localStorage after login", async ({ page }) => {
  75  |     await clearMailbox();
  76  |     await requestMagicLink(volunteerEmail);
  77  |     const msg = await waitForEmail(volunteerEmail);
  78  |     const magicUrl = extractMagicLink(msg);
  79  | 
  80  |     await page.goto(magicUrl);
  81  |     await page.waitForURL("**/events", { timeout: 8_000 });
  82  | 
  83  |     const token = await page.evaluate(() => localStorage.getItem("authToken"));
  84  |     expect(token).toBeTruthy();
  85  | 
  86  |     const role = await page.evaluate(() => localStorage.getItem("authRole"));
  87  |     expect(role).toBe("VOLUNTEER");
  88  |   });
  89  | });
  90  | 
  91  | test.describe("Magic-link login — admin routing", () => {
  92  |   test("administrator sees admin menu items when logged in", async ({ adminPage }) => {
  93  |     await adminPage.goto("/events");
  94  | 
  95  |     // Open the user menu and check for admin-only items
  96  |     await adminPage.getByRole("button", { name: /menu|account|settings/i }).first().click();
  97  |     await expect(adminPage.getByRole("link", { name: "Manage Events" })).toBeVisible();
  98  |     await expect(adminPage.getByRole("link", { name: "Manage Volunteers" })).toBeVisible();
  99  |   });
  100 | });
  101 | 
  102 | test.describe("Logout", () => {
  103 |   const AUTH_URL =
  104 |     process.env.NEXT_PUBLIC_GRAPHQL_AUTH_URL || "http://localhost:8080/graphql/auth";
  105 |   const VOLUNTEER_URL =
  106 |     process.env.NEXT_PUBLIC_GRAPHQL_VOLUNTEER_URL || "http://localhost:8080/graphql/volunteer";
  107 | 
  108 |   test("logout mutation invalidates the server session token", async ({
  109 |     volunteerToken,
  110 |   }) => {
  111 |     // Verify the token works before logout.
  112 |     const beforeRes = await fetch(VOLUNTEER_URL, {
  113 |       method: "POST",
  114 |       headers: {
  115 |         "Content-Type": "application/json",
  116 |         Authorization: `Bearer ${volunteerToken}`,
  117 |       },
  118 |       body: JSON.stringify({ query: "query { lookupValues { cities } }" }),
  119 |     });
  120 |     expect(beforeRes.status).toBe(200);
  121 | 
  122 |     // Call the logout mutation.
  123 |     const logoutRes = await fetch(AUTH_URL, {
  124 |       method: "POST",
  125 |       headers: { "Content-Type": "application/json" },
  126 |       body: JSON.stringify({
  127 |         query: `mutation Logout($token: String!) { logout(token: $token) { success } }`,
  128 |         variables: { token: volunteerToken },
  129 |       }),
  130 |     });
  131 |     const logoutJson = (await logoutRes.json()) as {
  132 |       data?: { logout?: { success: boolean } };
  133 |     };
  134 |     expect(logoutJson.data?.logout?.success).toBe(true);
  135 | 
  136 |     // The same token should now be rejected by the authenticated endpoint.
  137 |     const afterRes = await fetch(VOLUNTEER_URL, {
  138 |       method: "POST",
  139 |       headers: {
  140 |         "Content-Type": "application/json",
  141 |         Authorization: `Bearer ${volunteerToken}`,
  142 |       },
  143 |       body: JSON.stringify({ query: "query { lookupValues { cities } }" }),
  144 |     });
  145 |     expect(afterRes.status).toBe(401);
  146 |   });
  147 | 
  148 |   test("sign-out via UI clears session and redirects to /login", async ({
  149 |     volunteerPage,
  150 |   }) => {
  151 |     await volunteerPage.goto("/events");
  152 |     await expect(
  153 |       volunteerPage.getByRole("heading", { name: /volunteer events/i })
  154 |     ).toBeVisible({ timeout: 8_000 });
  155 | 
  156 |     // Open the UserMenu and click Sign Out.
> 157 |     await volunteerPage.getByRole("button", { name: /menu|account|settings/i }).first().click();
      |                                                                                         ^ Error: locator.click: Target page, context or browser has been closed
  158 |     await volunteerPage.getByRole("button", { name: /sign out/i }).click();
  159 | 
  160 |     // Should land on /login.
  161 |     await volunteerPage.waitForURL("**/login", { timeout: 5_000 });
  162 |     expect(volunteerPage.url()).toContain("/login");
  163 | 
  164 |     // localStorage is cleared — navigating to /events should redirect back to /login.
  165 |     await volunteerPage.goto("/events");
  166 |     await volunteerPage.waitForURL("**/login", { timeout: 5_000 });
  167 |     expect(volunteerPage.url()).toContain("/login");
  168 |   });
  169 | });
  170 | 
  171 | test.describe("Magic-link login — error cases", () => {
  172 |   test("unknown email shows 'No account found' with request-account option", async ({
  173 |     page,
  174 |   }) => {
  175 |     await page.goto("/login");
  176 |     await page.getByLabel("Email address").fill("nobody@definitely-not-real.test");
  177 |     await page.getByRole("button", { name: "Continue" }).click();
  178 | 
  179 |     await expect(
  180 |       page.getByRole("heading", { name: "No account found" })
  181 |     ).toBeVisible();
  182 |     await expect(
  183 |       page.getByRole("button", { name: "Request an Account" })
  184 |     ).toBeVisible();
  185 |   });
  186 | 
  187 |   test("invalid magic-link token shows sign-in failed page", async ({
  188 |     page,
  189 |   }) => {
  190 |     await page.goto("/auth/magic-link?token=totally-invalid-token");
  191 |     await expect(
  192 |       page.getByRole("heading", { name: "Sign-in failed" })
  193 |     ).toBeVisible();
  194 |     await expect(
  195 |       page.getByRole("link", { name: /new sign-in link/i })
  196 |     ).toBeVisible();
  197 |   });
  198 | 
  199 |   test("missing token in magic-link URL shows error", async ({ page }) => {
  200 |     await page.goto("/auth/magic-link");
  201 |     await expect(
  202 |       page.getByRole("heading", { name: "Sign-in failed" })
  203 |     ).toBeVisible();
  204 |     await expect(page.getByText(/no token/i)).toBeVisible();
  205 |   });
  206 | 
  207 |   test("unauthenticated user visiting /events is redirected to /login", async ({
  208 |     page,
  209 |   }) => {
  210 |     // Make sure localStorage is clear
  211 |     await page.goto("/");
  212 |     await page.evaluate(() => localStorage.clear());
  213 | 
  214 |     await page.goto("/events");
  215 |     await page.waitForURL("**/login", { timeout: 5_000 });
  216 |     expect(page.url()).toContain("/login");
  217 |   });
  218 | 
  219 |   test("account request form can be submitted for unknown email", async ({
  220 |     page,
  221 |   }) => {
  222 |     const newEmail = uniqueEmail("newrequest");
  223 | 
  224 |     await page.goto("/login");
  225 |     await page.getByLabel("Email address").fill(newEmail);
  226 |     await page.getByRole("button", { name: "Continue" }).click();
  227 |     await page.getByRole("button", { name: "Request an Account" }).click();
  228 | 
  229 |     // Fill in the request form
  230 |     await page.getByLabel("First name").fill("New");
  231 |     await page.getByLabel("Last name").fill("User");
  232 |     await page.getByRole("button", { name: "Submit Request" }).click();
  233 | 
  234 |     await expect(
  235 |       page.getByRole("heading", { name: "Request Submitted" })
  236 |     ).toBeVisible();
  237 |   });
  238 | });
  239 | 
```
Release notes
=============

- Update status-line even if no keys are typed

0.3.0
-----
on Sep.23,2021

- Fix the problem that the utf8-rune on the line boundary could not be drawn
- `w`: restore the last saved filename as the next saving
- `w`: show `canceled` instead of `^C` when ESCAPE key is pressed
- Display CR, LF, TAB with half width arrows
- Read data while waiting key typed
- Improve the internal data structure and be able to read more huge data
- Fix: the text color remained yellow even after the program ended

0.2.1
-----
on Jul 5,2021

- (#1) Fix the overflow that pointer to seek the top of the rune is decreased less than zero (Thx @spiegel-im-spiegel)
- If the cursor is not on utf8 sequences, print `(not utf8)`
- If the parameter is a directory, show error and quit immediately instead of hanging

0.2.0
-----
on Jul 5,2021

- Status line:
    - current rune's codepoint
    - changed/unchanged mark
- Implement key feature
    - p (paste 1 byte the rightside of the cursor)
    - P (paste 1 byte the leftside of the cursor)
    - a (append '\0' at the rightside of the cursor)
- Update library [go-readline-ny to v0.4.13](https://github.com/zetamatta/go-readline-ny/releases/tag/v0.4.13)

0.1.1
-----
on Dec 28,2020

- Did go mod init to fix the problem not able to build because the incompatibility of go-readline-ny between v0.2.6 and v0.2.8
- The binary executable of v0.1.0 has no problems.

0.1.0
-----
on Nov 8,2020

- The first version

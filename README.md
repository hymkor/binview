binview - Binary data viewer / editor
========================

![ScreenShot](./screenshot.png)

## Key Features

* **Fast startup with asynchronous loading**
  The viewer launches instantly and loads data in the background, allowing immediate interaction even with large files.

* **Supports both files and standard input**
  `binview` can read binary data not only from files but also from standard input, making it easy to use in pipelines.

* **Vi-style navigation**
  Navigation keys follow the familiar `vi` keybindings (`h`, `j`, `k`, `l`, etc.), allowing smooth movement for experienced users.  
(Note: File name input uses Emacs-style key bindings.)

* **Split-view with hex and character representations**
  The screen is divided approximately 2:1 between hexadecimal and character views. Supported encodings include UTF-8, UTF-16 (LE/BE), and the current Windows code page. You can switch encoding on the fly with key commands.

* **Smart decoding with character annotations**
  Multi-byte characters are visually grouped based on byte structure. Special code points such as BOMs and control characters (e.g., newlines) are annotated with readable names or symbols, making it easier to understand mixed binary/text content and debug encoding issues.

* **Minimal screen usage**
  `binview` only uses as many terminal lines as needed (1 line = 16 bytes), without occupying the full screen. This makes it easy to inspect or edit small binary data while still seeing the surrounding terminal output.

* **Cross-platform**
  Written in Go, `binview` runs on both Windows and Linux. It should also build and work on other Unix-like systems.

Install
--------

### Manual installation

Download the binary package from [Releases](https://github.com/hymkor/binview/releases) and extract the executable.

### Use "go install"

```
go install github.com/hymkor/binview@latest
```

### Use scoop-installer

```
scoop install https://raw.githubusercontent.com/hymkor/binview/master/binview.json
```

or

```
scoop bucket add hymkor https://github.com/hymkor/scoop-bucket
scoop install binview
```

Usage
-----

```
$ binview [FILES...]
```

or

```
$ cat FILE | binview
```

Key-binding
-----------

* q , ESCAPE
    * quit
* h , BACKSPACE , ARROW-LEFT , Ctrl-B
    * move the cursor left.
* j , ARROW-DOWN , Ctrl-N
    * move the cursor down.
* k , ARROW-UP , Ctrl-P
    * move the cursor up.
* l , SPACE , ARRIW-RIGHT , Ctrl-F
    * move the cursor right.
* 0(zero) , ^ , Ctrl-A
    * move the cursor to the top of the current line.
* $ , Ctrl-E
    * move the cursor to the tail of the current line.
* &lt;
    * move the cursor to the begin of the file.
* &gt; G
    * move thr cursor to the end of the file.
* r
    * replace one byte
* i
    * insert data (for example: `0xFF`,`U+0000`,`"string"`)
* a
    * append data (for example: `0xFF`,`U+0000`,`"string"`)
* x , DEL
    * delete and yank one byte on the cursor
* p
    * paste 1 byte the rightside of the cursor
* P
    * paste 1 byte the leftside of the cursor
* u
    * undo
* w
    * output to file
* &amp;
    * move the cursor to the address input
* ALT-U
    * Change the character encoding to UTF8 (default)
* ALT-A
    * Change the character encoding to ANSI, the current codepage (Windows-Only)
* ALT-L
    * Change the character encoding to UTF16LE
* ALT-B
    * Change the character encoding to UTF16BE

Release Note
============

- [English](/release_note_en.md)
- [Japanese](/release_note_ja.md)

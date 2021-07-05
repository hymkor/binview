binview - Binary data viewer / editor
========================

```
$ binview [FILES...]
```

or

```
$ binview < FILE
```

![ScreenShot](./screenshot.png)

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
    * insert '\0' on the cursor
* a
    * append '\0' at the rightside of the cursor
* x , DEL
    * delete and yank one byte on the cursor
* p
    * paste 1 byte the rightside of the cursor
* P
    * paste 1 byte the leftside of the cursor
* w
    * output to file

[Release Note](.\release_note.md)

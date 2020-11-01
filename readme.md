binview
=======

Binary file viewer

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
* h , ARROW-LEFT , Ctrl-B
    * move the cursor left.
* j , ARROW-DOWN , Ctrl-N
    * move the cursor down.
* k , ARROW-UP , Ctrl-P
    * move the cursor up.
* l , ARRIW-RIGHT , Ctrl-f
    * move the cursor right.
* 0(zero) , ^ , Ctrl-A
    * move the cursor to the top of the current line.
* $ , Ctrl-E
    * move the cursor to the tail of the current line.
* &lt;
    * move the cursor to the begin of the file.
* &gt;
    * move thr cursor to the end of the file.
* x
    * delete one byte on the cursor
* w
    * output to file

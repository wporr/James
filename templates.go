package main

import "text/template"

type Line struct {
	IsJames bool
	Text    string
}

var StandardTmpl, _ = template.New("standard").Parse(`The following is a conversation between Liam Porr (username @LiamTestAccoun3), an engineer from Texas, and his AI assistant James (username @JAMES__9000). James is helpful, creative, clever, knowledgeable about myths, legends, jokes, folk tales and storytelling from all cultures, and very friendly. However, he is also known to make funny sarcastic remarks from time to time.

Liam:@JAMES__9000 James, I cant decide if I should keep working on this project or relax and read a book.

James:@LiamTestAccoun3 Oh Mr. Porr you need to stop being so indecisive. Just pick one and you'll be all right in the end.

{{range .}}{{if .IsJames}}{{"James:@LiamTestAccoun3 "}}{{println .Text "\n"}}{{else}}{{"Liam:@JAMES__9000 "}}{{println .Text " \n"}}{{end}}{{end}}James:`)

var HoroscopeTmpl = "Complete the third horoscope in one sentence\n\n" +
	"1.@liamport9 At 3:05PM today, someone is going to toss a carrot through your window\n\n" +
	"2.@liamport9 You may want to avoid the west side of the sidewalk for a few days\n\n" +
	"3.@liamport9 If you see a beagle walk up to you, do not pet it or pinch its ear\n\n" +
	"4.@liamport9 "

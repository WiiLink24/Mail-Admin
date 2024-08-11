package main

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/WiiLink24/nwc24"
	"github.com/gin-gonic/gin"
	
)

var (
	InsertMail = `INSERT INTO mail (snowflake, data, sender, recipient, is_sent) VALUES ($1, $2, $3, $4, false)`
)

func SendMessage(c *gin.Context) {
	//fetch the message from the form
	subject := c.PostForm("subject")
	message := c.PostForm("message_content")
	recipient := c.PostForm("recipient")

	formatted_recipient := strings.ReplaceAll(recipient, "-", "")

	sender_address, err := mail.ParseAddress("w9999999900000000@rc24.xyz")
	if err != nil {
		fmt.Println(err)
	}

	recipient_address, err := mail.ParseAddress("w" + formatted_recipient + "@rc24.xyz")
	if err != nil {
		fmt.Println(err)
	}

	data := nwc24.NewMessage(sender_address, recipient_address	)
	data.SetSubject(subject)
	data.SetText(message, "utf-16be")
	
	content, err := data.ToString()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(content)
	}

	_, err = wiiMailPool.Exec(ctx, InsertMail, flakeNode.Generate(), content, "9999999900000000", formatted_recipient)
	if err != nil {
		fmt.Println(err)
	}

}

func CheckInboundOutbound(c *gin.Context) {
	/* number := c.Param("number") */
}

func ClearInbound(c *gin.Context) {
	
}
package main

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/WiiLink24/nwc24"
	"github.com/gin-gonic/gin"
)

func SendMessage(c *gin.Context) {
	//fetch the message from the form
	recipient_type := c.PostForm("recipient_type")
	subject := c.PostForm("subject")
	message := c.PostForm("message_content")
	recipient := c.PostForm("recipient")

	if recipient_type == "all" {
		recipient = "allusers@rc24.xyz"
	} else if recipient_type == "single" {
		formatted_number := strings.ReplaceAll(recipient, "-", "")
		recipient = "w" + formatted_number + "@rc24.xyz"
	}

	sender_address, err := mail.ParseAddress("w9999999900000000@rc24.xyz")
	if err != nil {
		fmt.Println(err)
	}

	recipient_address, err := mail.ParseAddress(recipient)
	if err != nil {
		fmt.Println(err)
	}

	data := nwc24.NewMessage(sender_address, recipient_address)
	data.SetSubject(subject)
	data.SetText(message, "utf-16be")
	
	content, err := data.ToString()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(content)
	}
}

func ClearInbound(c *gin.Context) {
	
}
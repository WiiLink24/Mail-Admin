package main

import (
	"fmt"
	"log"
	"net/mail"
	"strings"
	"unicode/utf16"

	"github.com/WiiLink24/nwc24"
	"github.com/gin-gonic/gin"
)

var (
    InsertMail = `INSERT INTO mail (snowflake, data, sender, recipient, is_sent) VALUES ($1, $2, $3, $4, false)`
)

func SendMessage(c *gin.Context) {
    //fetch the message from the form
    recipient_type := c.PostForm("recipient_type")
    subject := c.PostForm("subject")
    message := c.PostForm("message_content")
    recipient := c.PostForm("recipient")
    /* attachment := c.PostForm("attachment")

    //convert attachment to []byte
    attachment_data := []byte(attachment) */

    conv_message := utf16.Encode([]rune(message))
    message = nwc24.UTF16ToString(conv_message)

    formatted_recipient := strings.ReplaceAll(recipient, "-", "")

    sender_address, err := mail.ParseAddress("w9999999900000000@rc24.xyz")
    if err != nil {
        fmt.Println(err)
    }

    var recipient_address *mail.Address

    if recipient == "" && recipient_type == "all" {
        recipient_address, err = mail.ParseAddress("allusers@rc24.xyz")
        if err != nil {
            fmt.Println(err)
        }

    } else if recipient != "" && recipient_type == "single" {
        recipient_address, err = mail.ParseAddress("w" + formatted_recipient + "@rc24.xyz")
        if err != nil {
            fmt.Println(err)
        }
    } else {
        fmt.Println("Invalid recipient type")
    }

    //initialize the message
    data := nwc24.NewMessage(sender_address, recipient_address)
    data.SetSubject(subject)
    /* data.SetText(message, nwc24.UTF16BE) */
    data.SetBoundary(generateBoundary())
    data.SetContentType(nwc24.MultipartMixed)
    data.SetTag("X-Wii-MB-NoReply", "1")

    //create the multipart
    multipart := nwc24.NewMultipart()

    //now we append the data
    multipart.SetText(message, nwc24.UTF16BE)
    multipart.SetContentType(nwc24.PlainText)
/* 
    img_multipart := nwc24.NewMultipart()
    img_multipart.AddFile("attachment", attachment_data, nwc24.Jpeg) */

    //add the multipart to the message
    data.AddMultipart(multipart)

    content, err := nwc24.CreateMessageToSend(generateBoundary(), data)
    if err != nil {
        fmt.Println(err)
    } else {
        fmt.Println(content)
    }

    // Fetch the message from the form only if letter and thumbnail are uploaded
    letterFile, _ := c.FormFile("letter")
    thumbnailFile, _ := c.FormFile("thumbnail")

    if letterFile != nil || thumbnailFile != nil {
        uploadToGenerator(c, "letter", "letter.png")
        uploadToGenerator(c, "thumbnail", "thumbnail.png")

        audioFile, _, _ := c.Request.FormFile("audio")
        if audioFile != nil {
            uploadToGenerator(c, "audio", "sound.wav")
        }
        letterheadContent, _ := generateLetterhead()
		log.Printf("Letterhead content: %s", letterheadContent)
    } else {
        fmt.Println("No letter or thumbnail uploaded, skipping...")
    }

    if recipient_type == "all" {
        //insert null value for recipient
        _, err = wiiMailPool.Exec(ctx, InsertMail, flakeNode.Generate(), content, "9999999900000000", "")
    } else {
        _, err = wiiMailPool.Exec(ctx, InsertMail, flakeNode.Generate(), content, "9999999900000000", formatted_recipient)
    }
    if err != nil {
        fmt.Println(err)
    }
}


func CheckInboundOutbound(c *gin.Context) {
	/* number := c.Param("number") */
}

func ClearInbound(c *gin.Context) {

}

package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf16"
	"os"

	"github.com/WiiLink24/nwc24"
	"github.com/gin-gonic/gin"
)

var (
	InsertMail        = `INSERT INTO mail (snowflake, data, sender, recipient, is_sent) VALUES ($1, $2, $3, $4, false)`
	CheckRegistration = `SELECT EXISTS(SELECT 1 FROM accounts WHERE mlid = $1)`
)

func SendMessage(c *gin.Context) {
	//fetch the message from the form
	recipient_type := c.PostForm("recipient_type")
	subject := c.PostForm("subject")
	message := c.PostForm("message_content")
	recipient := c.PostForm("recipient")
	attachment, _ := c.FormFile("attachment")
	mii, _ := c.FormFile("mii")

	conv_message := utf16.Encode([]rune(message))
	message = nwc24.UTF16ToString(conv_message)

	formatted_recipient := strings.ReplaceAll(recipient, "-", "")

	//validations
	//check if the recipient is valid
	if !validateFriendCode(formatted_recipient) {
		c.HTML(http.StatusInternalServerError, "send_message.html", gin.H{
			"Title": "Send Message | WiiLink Mail",
			"Error": "This Wii Number is invalid (most likely a default Dolphin number).",
		})
	}

	//check if the recipient is registered
	var exists bool
	row, err := wiiMailPool.Query(ctx, CheckRegistration, formatted_recipient)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"Title": "Error | WiiLink Mail",
			"Error": "Couldn't query the database.",
		})
	}

	for row.Next() {
		err = row.Scan(&exists)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"Title": "Error | WiiLink Mail",
				"Error": "Couldn't scan the rows.",
			})
		}
	}

	if !exists {
		c.HTML(http.StatusInternalServerError, "send_message.html", gin.H{
			"Title": "Send Message | WiiLink Mail",
			"Error": "This Wii Number is not registered in the database.",
		})
	}

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
	data.SetBoundary(generateBoundary())
	data.SetContentType(nwc24.MultipartMixed)
	data.SetTag("X-Wii-MB-NoReply", "1")

	// Attach the Mii if it exists
	if mii != nil {
		// get the Mii data into base64 utf-16be
		mii_data := base64.StdEncoding.EncodeToString(encodeToUTF16BE(mii.Filename))
		data.SetTag("X-WiiFace", mii_data)
	} else {
		fmt.Println("No Mii uploaded, skipping...")
	}

	//create the text multipart
	text_multipart := nwc24.NewMultipart()

	//now we append the data
	text_multipart.SetText(message, nwc24.UTF16BE)
	text_multipart.SetContentType(nwc24.PlainText)

	//add the multipart to the message
	data.AddMultipart(text_multipart)

	// Attach the attachment image if it exists
	if err != nil {
		fmt.Println("Error retrieving the file:", err)
		return
	}

	if attachment != nil {
		attachmentMultipart := nwc24.NewMultipart()
		attachmentMultipart.SetContentType(nwc24.Jpeg)

		file, err := attachment.Open()
		if err != nil {
			fmt.Println("Error opening the file:", err)
			return
		}
		defer file.Close()

		attachmentBytes, err := io.ReadAll(file)
		if err != nil {
			fmt.Println("Error reading the file:", err)
			return
		}

		attachmentMultipart.AddFile("attachment", attachmentBytes, nwc24.Jpeg)
		data.AddMultipart(attachmentMultipart)
	} else {
		fmt.Println("No attachment uploaded, skipping...")
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

		// Include the letterhead in the message
		letterheadMultipart := nwc24.NewMultipart()
		letterheadMultipart.AddFile("letterhead", []byte(letterheadContent), nwc24.WiiMessageBoard)
		data.AddMultipart(letterheadMultipart)

	} else {
		fmt.Println("No letter or thumbnail uploaded, skipping...")
	}

	// Generate the message
	content, err := nwc24.CreateMessageToSend(generateBoundary(), data)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(content)

		// Write into txt file
		file, err := os.Create("generated_message.txt")
		if err != nil {
			fmt.Println("Error creating file:", err)
		} else {
			defer file.Close()
			_, err = file.WriteString(content)
			if err != nil {
				fmt.Println("Error writing to file:", err)
			}
		}
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

	c.Redirect(http.StatusTemporaryRedirect, "/send#success")
}

// idk if this works
func encodeToUTF16BE(s string) []byte {
	runes := utf16.Encode([]rune(s))
	buf := make([]byte, len(runes)*2)
	for i, r := range runes {
		buf[i*2] = byte(r >> 8)
		buf[i*2+1] = byte(r)
	}
	return buf
}

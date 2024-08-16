package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"unicode/utf16"

	"github.com/WiiLink24/nwc24"
	"github.com/gin-gonic/gin"
)

var (
	InsertMail        = `INSERT INTO mail (snowflake, data, sender, recipient, is_sent) VALUES ($1, $2, $3, $4, false)`
	CheckRegistration = `SELECT EXISTS(SELECT 1 FROM accounts WHERE mlid = $1)`
    CheckInboundOutbound = `SELECT (SELECT COUNT(*) FROM mail WHERE recipient = $1 AND is_sent = false) AS inbound_count, (SELECT COUNT(*) FROM mail WHERE sender = $1 AND is_sent = false) AS outbound_count`
    CheckOutbound = `SELECT COUNT(*) FROM mail WHERE sender = $1 AND is_sent = false`
    DeleteInbound = `DELETE FROM mail WHERE recipient = $1 AND is_sent = false`
    DeleteOutbound = `DELETE FROM mail WHERE sender = $1 AND is_sent = false`
	DeleteAccount = `DELETE FROM accounts WHERE mlid = $1`
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
		_, err = encryptMessage(content)
		if err != nil {
			fmt.Println(err)
		}
	} else {
		_, err = wiiMailPool.Exec(ctx, InsertMail, flakeNode.Generate(), content, "9999999900000000", formatted_recipient)
	}
	if err != nil {
		fmt.Println(err)
	}

	c.Redirect(http.StatusTemporaryRedirect, "/send#success")
}

func CheckInOutMessages(c *gin.Context) {
    wiiNumber := c.PostForm("wii_number")

    formatted_number := strings.ReplaceAll(wiiNumber, "-", "")

    if !validateFriendCode(formatted_number) {
        c.HTML(http.StatusInternalServerError, "inbound.html", gin.H{
            "Error": "This Wii Number is invalid (most likely a default Dolphin number).",
        })
    }

    var exists bool
	row, err := wiiMailPool.Query(ctx, CheckRegistration, formatted_number)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"Error": "Couldn't query the database.",
		})
	}

	for row.Next() {
		err = row.Scan(&exists)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"Error": "Couldn't scan the rows.",
			})
		}
	}

	if !exists {
		c.HTML(http.StatusInternalServerError, "send_message.html", gin.H{
			"Error": "This Wii Number is not registered in the database.",
		})
	}

    var inbound, outbound int
    rows, err := wiiMailPool.Query(ctx, CheckInboundOutbound, formatted_number)
    if err != nil {
        c.HTML(http.StatusInternalServerError, "error.html", gin.H{
            "Error": "Couldn't query the database.",
        })
    }

    for rows.Next() {
        err = rows.Scan(&inbound, &outbound)
        if err != nil {
            c.HTML(http.StatusInternalServerError, "error.html", gin.H{
                "Error": "Couldn't scan the rows.",
            })
        }
    }

    c.HTML(http.StatusOK, "inbound.html", gin.H{
        "Title": "Check Messages | WiiLink Mail",
        "Inbound": inbound,
        "Outbound": outbound,
    })

}

func DeleteMessages(c *gin.Context) {
    action_type := c.PostForm("type")
    wiiNumber := c.PostForm("wii_number")

    formatted_number := strings.ReplaceAll(wiiNumber, "-", "")

    if !validateFriendCode(formatted_number) {
        c.HTML(http.StatusInternalServerError, "inbound.html", gin.H{
            "Error": "This Wii Number is invalid (most likely a default Dolphin number).",
        })
    }

    var exists bool
    row, err := wiiMailPool.Query(ctx, CheckRegistration, formatted_number)
    if err != nil {
        c.HTML(http.StatusInternalServerError, "error.html", gin.H{
            "Error": "Couldn't query the database.",
        })
    }

    for row.Next() {
        err = row.Scan(&exists)
        if err != nil {
            c.HTML(http.StatusInternalServerError, "error.html", gin.H{
                "Error": "Couldn't scan the rows.",
            })
        }
    }
    
    if !exists {
        c.HTML(http.StatusInternalServerError, "send_message.html", gin.H{
            "Error": "This Wii Number is not registered in the database.",
        })
    }

    if action_type == "inbound" {
        _, err = wiiMailPool.Exec(ctx, DeleteInbound, formatted_number)
        if err != nil {
            fmt.Println(err)
        }
    } else if action_type == "outbound" {
        _, err = wiiMailPool.Exec(ctx, DeleteOutbound, formatted_number)
        if err != nil {
            fmt.Println(err)
        }
    }

    c.Redirect(http.StatusTemporaryRedirect, "/clear#success")

}

func checkIsValidNumber(c *gin.Context) {
	wiiNumber := c.PostForm("wii_number")

	formatted_number := strings.ReplaceAll(wiiNumber, "-", "")

	if !validateFriendCode(formatted_number) {
		c.HTML(http.StatusOK, "misc.html", gin.H{
			"Title": "Miscellaneous | WiiLink Mail",
			"Result": "This Wii Number is invalid. It could be either a default Dolphin number, or a mistyped number.",
		})
	} else {
		c.HTML(http.StatusOK, "misc.html", gin.H{
			"Title": "Miscellaneous | WiiLink Mail",
			"Result": "This Wii Number is valid.",
		})
	}
}

func checkIsRegistered(c *gin.Context) {
	wiiNumber := c.PostForm("wii_number")

	formatted_number := strings.ReplaceAll(wiiNumber, "-", "")

	var exists bool
	row, err := wiiMailPool.Query(ctx, CheckRegistration, formatted_number)
	if err != nil {
		c.HTML(http.StatusOK, "misc.html", gin.H{
			"Title": "Miscellaneous | WiiLink Mail",
			"Error": "Couldn't query the database.",
		})
	}

	for row.Next() {
		err = row.Scan(&exists)
		if err != nil {
			c.HTML(http.StatusOK, "misc.html", gin.H{
				"Title": "Miscellaneous | WiiLink Mail",
				"Error": "Couldn't scan the rows.",
			})
		}
	}

	if exists {
		c.HTML(http.StatusOK, "misc.html", gin.H{
			"Title": "Miscellaneous | WiiLink Mail",
			"Result": "This Wii Number is registered in the database.",
		})
	} else {
		c.HTML(http.StatusOK, "misc.html", gin.H{
			"Title": "Miscellaneous | WiiLink Mail",
			"Result": "This Wii Number is not registered.",
		})
	}
}

func RemoveAccount(c *gin.Context) {
	wiiNumber := c.PostForm("wii_number")

	formatted_number := strings.ReplaceAll(wiiNumber, "-", "")

	if !validateFriendCode(formatted_number) {
		c.HTML(http.StatusOK, "misc.html", gin.H{
			"Title": "Miscellaneous | WiiLink Mail",
			"Error": "This Wii Number is invalid (most likely a default Dolphin number).",
		})
	}

	var exists bool
	row, err := wiiMailPool.Query(ctx, CheckRegistration, formatted_number)
	if err != nil {
		c.HTML(http.StatusOK, "misc.html", gin.H{
			"Title": "Miscellaneous | WiiLink Mail",
			"Error": "Couldn't query the database.",
		})
	}

	for row.Next() {
		err = row.Scan(&exists)
		if err != nil {
			c.HTML(http.StatusOK, "misc.html", gin.H{
				"Title": "Miscellaneous | WiiLink Mail",
				"Error": "Couldn't scan the rows.",
			})
		}
	}

	if !exists {
		c.HTML(http.StatusOK, "misc.html", gin.H{
			"Title": "Miscellaneous | WiiLink Mail",
			"Error": "This Wii Number is not registered in the database.",
		})
	} else {
		_, err = wiiMailPool.Exec(ctx, DeleteAccount, formatted_number)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"Title": "Error | WiiLink Mail",
				"Error": "Couldn't delete the account.",
			})
		}

		c.HTML(http.StatusOK, "misc.html", gin.H{
			"Title": "Miscellaneous | WiiLink Mail",
			"Result": "The account has been removed.",
		})
	}
}


package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/wii-tools/arclib"
	"image/jpeg"
	"io"
	"net/http"
	"net/mail"
	"os"
	"unicode/utf16"

	"github.com/WiiLink24/nwc24"
	"github.com/gin-gonic/gin"
)

func SendMessage(c *gin.Context) {
	//fetch the message from the form
	subject := c.PostForm("subject")
	message := c.PostForm("message_content")
	attachment, _ := c.FormFile("attachment")
	mii, _ := c.FormFile("mii")
	language := c.PostForm("language")

	message = nwc24.UTF16ToString(utf16.Encode([]rune(message)))

	from, err := mail.ParseAddress("w9999999900000000@rc24.xyz")
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"Error": err.Error(),
		})
		return
	}

	to, err := mail.ParseAddress("allusers@rc24.xyz")
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"Error": err.Error(),
		})
		return
	}

	// Initialize the message
	// Use distinct boundaries for the outer message and the inner
	outerBoundary := generateBoundary()
	innerBoundary := generateBoundary()

	msg := nwc24.NewMessage(from, to)
	msg.SetBoundary(innerBoundary)
	msg.SetContentType(nwc24.MultipartMixed)
	msg.SetTag("X-Wii-MB-NoReply", "1")
	msg.SetTag("X-Wii-MB-OptOut", "1")
	msg.SetTag("X-Wii-AltName", nwc24.Base64Encode(UTF16ToBytes(utf16.Encode([]rune(subject)))))

	// Attach the Mii if it exists
	if mii != nil {
		// First open the Mii
		miiFp, err := mii.Open()
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"Error": err.Error(),
			})
			return
		}

		defer miiFp.Close()
		miiBytes, err := io.ReadAll(miiFp)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"Error": err.Error(),
			})
			return
		}

		// Steps must be taken to make sure this doesn't fail.
		// First, strip the CRC16 checksum then encode in Base64.
		miiB64 := base64.StdEncoding.EncodeToString(miiBytes[:74])

		// Next, we need to format the string correctly.
		// First line (the header) must be 67 characters long including the key `X-WiiFace: `.
		// It is followed by a carriage return, a space, then the remaining base64 characters.
		keyLen := len("X-WiiFace: ")
		miiB64 = miiB64[:67-keyLen] + "\r\n " + miiB64[67-keyLen:]

		msg.SetTag("X-WiiFace", miiB64)
		msg.SetTag("X-Wii-AppId", "2-48414541-0001")
		msg.SetTag("X-Wii-Cmd", "00044001")
	} else {
		fmt.Println("No Mii uploaded, skipping...")
	}

	// Next, create the text section.
	textPart := nwc24.NewMultipart()
	textPart.SetText(message, nwc24.UTF16BE)

	// Add the text part to the message
	msg.AddMultipart(textPart)

	// Attach the attachment image if it exists
	if attachment != nil {
		attachmentMultipart := nwc24.NewMultipart()
		attachmentMultipart.SetContentType(nwc24.Jpeg)

		file, err := attachment.Open()
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"Error": err.Error(),
			})
			return
		}

		// We have to convert to baseline JPEG.
		decodedImage, err := resize(file)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"Error": err.Error(),
			})
			return
		}

		var jpegEncoded bytes.Buffer
		err = jpeg.Encode(bufio.NewWriter(&jpegEncoded), decodedImage, nil)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"Error": err.Error(),
			})
			return
		}

		attachmentMultipart.AddFile("image.jpg", jpegEncoded.Bytes(), nwc24.Jpeg)
		msg.AddMultipart(attachmentMultipart)
	} else {
		fmt.Println("No attachment uploaded, skipping...")
	}

	// Fetch the message from the form only if letter and thumbnail are uploaded
	letterFile, _ := c.FormFile("letter")
	thumbnailFile, _ := c.FormFile("thumbnail")

	if letterFile != nil || thumbnailFile != nil {
		letterFp, err := letterFile.Open()
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"Error": err.Error(),
			})
			return
		}

		// Make the letter images
		defer letterFp.Close()
		letterArc, err := makeLetterImages(letterFp)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"Error": err.Error(),
			})
			return
		}

		// Now the thumbnail image
		thumbnailFp, err := thumbnailFile.Open()
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"Error": err.Error(),
			})
			return
		}

		defer thumbnailFp.Close()
		thumbnailArc, err := makeThumbnail(thumbnailFp)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"Error": err.Error(),
			})
			return
		}

		// Finally we have to combine the two into one archive.
		letterHeadArc, err := arclib.Load(letterHeadArchiveBase)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"Error": err.Error(),
			})
			return
		}

		letterHeadArc.RootRecord.WriteFile("letter_LZ.bin", letterArc)
		letterHeadArc.RootRecord.WriteFile("thumbnail_LZ.bin", thumbnailArc)

		letterHeadBytes, err := letterHeadArc.Save()
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{
				"Error": err.Error(),
			})
			return
		}

		letterheadMultipart := nwc24.NewMultipart()
		letterheadMultipart.AddFile("letterhead.arc", letterHeadBytes, nwc24.WiiMessageBoard)
		msg.AddMultipart(letterheadMultipart)

		// TODO: Audio support
	} else {
		fmt.Println("No letter or thumbnail uploaded, skipping...")
	}

	content, err := nwc24.CreateMessageToSend(outerBoundary, msg)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"Error": err.Error(),
		})
		return
	}

	// Make directory and file
	err = os.MkdirAll(fmt.Sprintf("%s/%s", config.AssetsPath, language), 0775)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"Error": err.Error(),
		})
	}

	err = os.WriteFile(fmt.Sprintf("%s/%s/announcement", config.AssetsPath, language), []byte(content), 0644)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"Error": err.Error(),
		})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/send#success")
}

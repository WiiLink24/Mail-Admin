package main

import (
	"bytes"
	"fmt"
	"github.com/WiiLink24/nwc24"
	"github.com/gin-gonic/gin"
	"io"
	"io/ioutil"
	"log"
	"net/mail"
	"os"
	"os/exec"
	"strings"
)

var (
	InsertMail = `INSERT INTO mail (snowflake, data, sender, recipient, is_sent) VALUES ($1, $2, $3, $4, false)`
)

func SendMessage(c *gin.Context) {
	//fetch the message from the form
	subject := c.PostForm("subject")
	message := c.PostForm("message_content")
	recipient := c.PostForm("recipient")
	attachment := c.PostForm("attachment")

	//convert attachment to []byte
	attachment_data := []byte(attachment)

	formatted_recipient := strings.ReplaceAll(recipient, "-", "")

	sender_address, err := mail.ParseAddress("w9999999900000000@rc24.xyz")
	if err != nil {
		fmt.Println(err)
	}

	recipient_address, err := mail.ParseAddress("w" + formatted_recipient + "@rc24.xyz")
	if err != nil {
		fmt.Println(err)
	}

	//initialize the message
	data := nwc24.NewMessage(sender_address, recipient_address)
	data.SetSubject(subject)
	data.SetContentType(nwc24.MultipartMixed)
	data.SetBoundary(generateBoundary())
	data.SetTag("X-Wii-MB-NoReply", "1")
	data.SetTag("X-Wii-AppId", "2-48414541-0001")
	data.SetTag("X-Wii-Cmd", "00044001")

	//create the multipart
	multipart := nwc24.NewMultipart()

	//now we append the data
	multipart.SetText(message, nwc24.UTF16BE)
	multipart.SetContentType(nwc24.PlainText)

	img_multipart := nwc24.NewMultipart()
	img_multipart.AddFile("attachment", attachment_data, nwc24.Jpeg)

	//add the multipart to the message
	data.AddMultipart(multipart, img_multipart)

	content, err := data.ToString()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(content)
	}

	// Fetch the message from the form only if letter and thumbnail are uploaded
	letterFile, _, _ := c.Request.FormFile("letter")
	thumbnailFile, _, _ := c.Request.FormFile("thumbnail")

	if letterFile != nil || thumbnailFile != nil {
		uploadToGenerator(c, "letter", "letter.png")
		uploadToGenerator(c, "thumbnail", "thumbnail.png")

		audioFile, _, _ := c.Request.FormFile("audio")
		if audioFile != nil {
			uploadToGenerator(c, "audio", "sound.wav")
		}
		generateLetterhead()
	} else {
		fmt.Println("No letter or thumbnail uploaded, skipping...")
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

func uploadToGenerator(c *gin.Context, source string, destination string) {
	file, header, err := c.Request.FormFile(source)
	if err != nil {
		log.Printf("Failed to get uploaded file: %v", err)
		return
	}
	defer file.Close()

	// Get the filename
	filename := header.Filename
	fmt.Println("Received file:", filename)

	// Ensure the generator/input directory exists
	err = os.MkdirAll("generator/input", os.ModePerm)
	if err != nil {
		log.Printf("Failed to create directory: %v", err)
		return
	}

	// Create a file for the letterhead
	out, err := os.Create("generator/input/" + destination)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	// Write the uploaded file to the created file
	_, err = io.Copy(out, file)
	if err != nil {
		log.Printf("Failed to write letterhead data: %v", err)
	}
}

func generateLetterhead() (string, error) {
	var out bytes.Buffer
	var stderr bytes.Buffer

	// Run generate.sh
	cmd := exec.Command("sh", "generator/generate.sh")
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	log.Printf("Running command and waiting for it to finish...")
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run generate.sh: %v: %s", err, stderr.String())
	}

	// Read the data from generator/output/letterhead.txt
	letterheadData, err := ioutil.ReadFile("generator/output/letterhead.txt")
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	// Convert the data to a string and store it in a variable
	letterheadContent := string(letterheadData)
	log.Printf("Letterhead content: %v", letterheadContent)
	return letterheadContent, nil
}

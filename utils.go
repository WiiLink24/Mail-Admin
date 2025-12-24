package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"time"
	"unicode/utf16"

	"github.com/SketchMaster2001/libwc24crypt"
	"github.com/WiiLink24/nwc24"
	"github.com/gin-gonic/gin"
)

var (
	key = []byte{0xdb, 0xfc, 0xe7, 0x34, 0x51, 0xda, 0xbd, 0xf3, 0xf4, 0x81, 0x37, 0xe5, 0xed, 0x00, 0xb6, 0xd2}
	iv  = []byte{70, 70, 20, 40, 143, 110, 36, 6, 184, 107, 135, 239, 96, 45, 80, 151}
)

func generateBoundary() string {
	return fmt.Sprintf("%s-%d", time.Now().Format("20060102150405"), rand.Int63())
}

func validateFriendCode(strId string) bool {
	if len(strId) != 16 {
		// All Wii Numbers are 16 characters long.
		return false
	}

	id, err := strconv.ParseInt(strId, 10, 64)
	if err != nil {
		// Not an integer value, therefore not an ID
		return false
	}

	wiiNumber := nwc24.LoadWiiNumber(uint64(id))
	if !wiiNumber.CheckWiiNumber() {
		// Invalid Wii Number (crc is invalid)
		return false
	}

	return !(wiiNumber.GetHollywoodID() == 0x0403AC68)
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

	// Read the data from generator/output/letterhead.arc
	letterheadData, err := ioutil.ReadFile("generator/output/letterhead.arc")
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	// Convert the data to a string and store it in a variable
	letterheadContent := string(letterheadData)
	return letterheadContent, nil
}

func encodeToUTF16BE(s string) []byte {
	runes := utf16.Encode([]rune(s))
	buf := make([]byte, len(runes)*2)
	for i, r := range runes {
		buf[i*2] = byte(r >> 8)
		buf[i*2+1] = byte(r)
	}
	return buf
}

func encryptMessage(message string) (string, error) {
	rsa, err := os.ReadFile(GetConfig().AssetsPath + "/cmoc.pem")
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	enc, err := libwc24crypt.EncryptWC24([]byte(message), key, iv, rsa)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt message: %v", err)
	}

	return "", os.WriteFile(fmt.Sprintf("%s/output/annoucement.bin", GetConfig().AssetsPath), enc, 0664)

}

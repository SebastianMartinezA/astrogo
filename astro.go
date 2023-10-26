package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// A struct to hold the response payload
type Payload struct {
	Payload struct {
		FiatCurrency   string  `json:"fiat_currency"`
		CryptoCurrency string  `json:"crypto_currency"`
		FiatAmount     float64 `json:"fiat_amount"`
		UsdAmount      float64 `json:"usd_amount"`
		CryptoAmount   float64 `json:"crypto_amount"`
		Price          float64 `json:"price"`
	} `json:"payload"`
}

type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("Unkown fromServer")
		}
	}
	return nil, nil
}

func sendEmail(message string, rate float64) {
	auth := LoginAuth(os.Getenv("SENDER_EMAIL"), os.Getenv("SENDER_PASSWORD"))

	emailList, err := getEmailListFromFile("email_list.txt")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(emailList)

	subject := fmt.Sprintf("Subject: Alert USDT rate dropped to $%.2f ARS \n\n", rate)
	msg := []byte(subject + message)

	for _, recipient := range emailList {
		err := smtp.SendMail("smtp.gmail.com:587", auth, "from@example.com", []string{recipient}, msg)
		if err != nil {
			log.Println("Failed to send email to", recipient, "Error:", err)
		} else {
			fmt.Println("Email sent to", recipient)
		}
	}
}

func getEmailListFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var emailList []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		emailList = append(emailList, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return emailList, nil
}

// A function to make a GET request and parse the response
func getRequest(desiredRate float64) error {
	url := "https://app-api.astropay.com/v1/assets/markets/exchanges?fiatCurrency=ARS&cryptoCurrency=USDT&fiatAmount=10000&operation=BUY"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Cookie", os.Getenv("COOKIE"))

	req.Header.Set("Platform", "WEB")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	//fmt.Println("Body: ", string(body))

	var payload Payload
	err = json.Unmarshal(body, &payload)
	if err != nil {
		return err
	}

	currentTime := time.Now()
	timeFormat := "15:04:05"
	formattedTime := currentTime.Format(timeFormat)

	actualRate := payload.Payload.FiatAmount / payload.Payload.CryptoAmount
	fmt.Printf("\n%v Rate: %v\n", formattedTime, actualRate)

	if actualRate < desiredRate {
		message := fmt.Sprintf("The rate of ARS/USDT is %f, which is less than %f.", actualRate, desiredRate)
		fmt.Println(message)
		sendEmail(message, actualRate)
	}
	return nil
}

func setCookie() {
	file, err := os.Open("config.txt")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)

		if len(parts) != 2 {
			fmt.Println("Invalid line:", line)
			continue
		}

		key, value := parts[0], parts[1]

		os.Setenv(key, value)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}
}

func setUp() {
	setCookie()
}

func main() {

	setUp()

	var desiredRate float64

	fmt.Print("Enter desired rate: ")
	_, err := fmt.Scanf("%v", &desiredRate)

	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}

	fmt.Scanln()
	for {
		err := getRequest(desiredRate)
		if err != nil {
			fmt.Println("Error getting request:", err)
		}
		time.Sleep(20 * time.Second)
	}
}

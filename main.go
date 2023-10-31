package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

type StoredListing struct {
	Data json.RawMessage `json:"data"`
}

var listings []StoredListing
var currentID int

func main() {
	r := gin.Default()

	r.POST("/data", handleData)
	r.GET("/get", handleGet)

	port := ":" + os.Getenv("PORT")
	if err := r.Run(port); err != nil {
		panic(fmt.Sprintf("Failed to start server on port %s: %v", port, err))
	}
}

func handleData(c *gin.Context) {
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error reading request body"})
		return
	}

	// listing := StoredListing{
	// 	Data: body,
	// }
	// listings = append(listings, listing)
	// currentID++

	// Transform and send the data to the other server
	go sendDataToOtherServer(body)

	c.JSON(http.StatusOK, gin.H{"message": "Data received and stored"})
}

func sendDataToOtherServer(data []byte) {
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		fmt.Println("Error unmarshaling data:", err)
		return
	}

	keyBoldMap := jsonData["KeyBoldMap"].(map[string]interface{})

	transaction := "unknown"
	if url, ok := jsonData["URL"].(string); ok {
		if strings.Contains(url, "for-sale") {
			transaction = "sale"
		} else if strings.Contains(url, "for-lease") {
			transaction = "lease"
		}
	}

	transformedData := map[string]interface{}{
		"URL":         jsonData["URL"],
		"Location":    jsonData["Name"],
		"Photo":       jsonData["Photo"],
		"Asset":       keyBoldMap["Property Type"],
		"Size":        keyBoldMap["Building Size"],
		"Price":       keyBoldMap["Asking Price"],
		"Latitude":    "",
		"Longitude":   "",
		"Transaction": transaction,
		"LeaseRate":   keyBoldMap["Base Rent"],
		"State":       "AB",
	}

	transformedJSON, err := json.Marshal(transformedData)
	if err != nil {
		fmt.Println("Error marshaling transformed data:", err)
		return
	}

	url := "https://jsonserver-production-0d88.up.railway.app/add"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(transformedJSON))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Response status:", resp.Status)
}

func handleGet(c *gin.Context) {
	flattenedListings := make([]map[string]interface{}, 0)

	for _, listing := range listings {
		var data map[string]interface{}
		if err := json.Unmarshal(listing.Data, &data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unmarshaling data"})
			return
		}

		flattenedData := make(map[string]interface{})
		flattenedData["URL"] = data["URL"]
		flattenedData["Name"] = data["Name"]
		flattenedData["Photo"] = data["Photo"]
		for key, value := range data["KeyBoldMap"].(map[string]interface{}) {
			flattenedData[key] = value
		}

		flattenedListings = append(flattenedListings, flattenedData)
	}

	c.IndentedJSON(http.StatusOK, flattenedListings)
}

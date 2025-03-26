package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/translate"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/net/html"
)

// HTMLRequest represents the input JSON structure.
type HTMLRequest struct {
	SourceLanguage string `json:"SourceLanguage"`
	TargetLanguage string `json:"TargetLanguage"`
	HTML           string `json:"html"`
}

// HTMLResponse is the output JSON structure.
type HTMLResponse struct {
	TranslatedHTML string `json:"translatedHtml"`
}

var translateClient *translate.Client

func main() {
	e := echo.New()
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))

	// Set up AWS Translate client.
	region := "us-east-1"
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		fmt.Println("Error loading AWS config:", err)
		return
	}
	translateClient = translate.NewFromConfig(cfg)

	// Serve static home page (if needed).
	e.GET("/", homePage)
	// New endpoint to translate HTML content.
	e.POST("/translateHtml", translateHTMLHandler)

	fmt.Println("Server started on :3001")
	e.Start(":3001")
}

// homePage serves a static HTML file.
func homePage(c echo.Context) error {
	return c.File("index.html")
}

// translateHTMLHandler handles translation requests for full HTML pages.
func translateHTMLHandler(c echo.Context) error {
	var req HTMLRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid request")
	}

	translatedHTML, err := translateHTMLContent(req.HTML, req.SourceLanguage, req.TargetLanguage)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, fmt.Sprintf("Translation error: %v", err))
	}

	return c.JSON(http.StatusOK, HTMLResponse{TranslatedHTML: translatedHTML})
}

// translateHTMLContent parses the HTML, translates text nodes, and returns the modified HTML.
func translateHTMLContent(htmlStr, sourceLang, targetLang string) (string, error) {
	// Parse the HTML string into a node tree.
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", err
	}

	// Traverse the tree and translate text nodes.
	err = traverseAndTranslate(doc, sourceLang, targetLang)
	if err != nil {
		return "", err
	}

	// Render the modified tree back to HTML.
	var buf bytes.Buffer
	if err = html.Render(&buf, doc); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// traverseAndTranslate recursively traverses nodes, translating text nodes that aren't in <script> or <style>.
func traverseAndTranslate(n *html.Node, sourceLang, targetLang string) error {
	// Check if the current node is a text node and its parent is not a script or style tag.
	if n.Type == html.TextNode && n.Parent != nil {
		if n.Parent.Data != "script" && n.Parent.Data != "style" {
			trimmed := strings.TrimSpace(n.Data)
			if trimmed != "" {
				translated, err := translateText(trimmed, sourceLang, targetLang)
				if err != nil {
					return err
				}
				// Replace the text with the translated version while preserving surrounding whitespace.
				n.Data = strings.Replace(n.Data, trimmed, translated, 1)
			}
		}
	}
	// Process child nodes.
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if err := traverseAndTranslate(child, sourceLang, targetLang); err != nil {
			return err
		}
	}
	return nil
}

// translateText calls AWS Translate for the given text.
func translateText(text, sourceLang, targetLang string) (string, error) {
	input := &translate.TranslateTextInput{
		SourceLanguageCode: aws.String(sourceLang),
		TargetLanguageCode: aws.String(targetLang),
		Text:               aws.String(text),
	}
	result, err := translateClient.TranslateText(context.TODO(), input)
	if err != nil {
		return "", err
	}
	return *result.TranslatedText, nil
}

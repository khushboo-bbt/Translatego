package main

import (
    "context"
    "fmt"
    "net/http"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/translate"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

type Request struct {
    SourceLanguage string `json:"SourceLanguage"`
    TargetLanguage string `json:"TargetLanguage"`
    Text           string `json:"Text"`
}

type Response struct {
    TranslatedText string `json:"TranslatedText"`
}

func main() {
    e := echo.New()
    e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
        Format: "method=${method}, uri=${uri}, status=${status}\n",
    }))

    // Serve your HTML page
    e.GET("/", homePage)

    // Handle translation request
    e.POST("/translate", translateAndWrite)

    fmt.Println("Server started on :3001")
    e.Start(":3001")
}

func homePage(c echo.Context) error {
    return c.File("index.html")
}

func translateAndWrite(c echo.Context) error {
    region := "us-east-1"

    // Load AWS configuration
    cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
    if err != nil {
        fmt.Println("Couldn't load default configuration. Have you set up your AWS account?")
        fmt.Println(err)
        return c.JSON(http.StatusInternalServerError, "AWS configuration error")
    }

    translateClient := translate.NewFromConfig(cfg)

    var req Request
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, "Invalid request")
    }

    response, err := translateClient.TranslateText(context.TODO(), &translate.TranslateTextInput{
        SourceLanguageCode: aws.String(req.SourceLanguage),
        TargetLanguageCode: aws.String(req.TargetLanguage),
        Text:               aws.String(req.Text),
    })
    if err != nil {
        return c.JSON(http.StatusInternalServerError, "Translation error")
    }

    return c.JSON(http.StatusOK, Response{
        TranslatedText: *response.TranslatedText,
    })
}

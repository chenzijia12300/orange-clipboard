package client

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"golang.design/x/clipboard"
	"orange-clipboard/client/conf"
	"orange-clipboard/common/resource"
)

type ReadMessageHandler func([]byte) bool

type MessageListener func(MessageContainer) bool

var previousMessage string

var messageListenerList []MessageListener

func InitClipboard() error {
	err := clipboard.Init()
	if err != nil {
		resource.Logger.Error("初始化剪贴板组件失败:", zap.Error(err))
		return err
	}
	ListenClipboardText()
	ListenClipboardImage()
	return nil
}

// 监听剪贴板文本变动
func ListenClipboardText() {
	ctx := context.Background()
	textCh := clipboard.Watch(ctx, clipboard.FmtText)
	go func() {
		for messageBytes := range textCh {
			message := string(messageBytes)
			if needFilter(message) {
				continue
			}
			resource.Logger.Info("剪贴板文本信息:", zap.String("message", message))
			//secretData := Encrypt(clipboardConfig.SecretKey, messageBytes)
			secretData := messageBytes
			container := MessageContainer{
				Type: conf.NORMAL,
				Data: secretData,
			}
			for _, fun := range messageListenerList {
				fun(container)
			}
		}
	}()
}

func needFilter(message string) bool {
	if message == previousMessage {
		return true
	}
	return false
}

func WriteClipboard(secretData []byte) bool {
	//data := Decrypt(clipboardConfig.SecretKey, secretData)
	data := secretData
	previousMessage = string(data)
	resource.Logger.Info("写入剪贴板文本信息", zap.String("message", previousMessage))
	clipboard.Write(clipboard.FmtText, data)
	return true
}

func ListenClipboardImage() {
	ctx := context.Background()
	imgCh := clipboard.Watch(ctx, clipboard.FmtImage)
	go func() {
		for messageBytes := range imgCh {
			fmt.Println("剪贴板图像信息:", len(messageBytes))
			// TODO 发送信息websocket
		}
	}()
}

/*
	添加剪贴板文本监听器
*/

func AddMessageListener(listener MessageListener) {
	messageListenerList = append(messageListenerList, listener)
}

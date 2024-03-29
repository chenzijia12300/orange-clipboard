package client

import (
	"context"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"net/http"
	"orange-clipboard/client/conf"
	"orange-clipboard/common/resource"
	"time"
)

type ConnectAction int

type MessageContainer struct {
	Type int
	Data []byte
}

var (
	messageCh        chan MessageContainer
	connectErrorFlag bool

	WriteMessageCh chan<- MessageContainer
	CloseCh        chan struct{}
)

const (
	pongWait        = 30 * time.Second
	pingPeriod      = pongWait * 9 / 10
	reConnectPeriod = 5 * time.Second
)

func InitConnectServer(ctx context.Context) {
	messageCh = make(chan MessageContainer)
	WriteMessageCh = messageCh
	serverUrl := conf.GlobalConfig.ServerUrl
	header := http.Header{}
	header.Add(conf.SystemName, conf.GlobalConfig.SystemName)
	header.Add(conf.DeviceName, conf.GlobalConfig.DeviceName)
	go ReConnectServer(ctx)
	conn, _, err := websocket.DefaultDialer.Dial(serverUrl, header)
	if err != nil {
		resource.Logger.Error("连接服务器失败", zap.String("serverUrl", conf.GlobalConfig.ServerUrl), zap.Error(err))
		go SetConnectErrorFlag(true)
		return
	}
	WriteServerMessage(conn)
	go ReadServerMessage(conn, WriteClipboard)
	go SetConnectErrorFlag(false)
	go CloseServer(conn)
}

func ReadServerMessage(conn *websocket.Conn, readHandler ReadMessageHandler) {
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(s string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil && messageType == conf.CANCEL {
			resource.Logger.Info("服务器断开连接",
				zap.String("serverUrl", conn.RemoteAddr().String()))
			SetConnectErrorFlag(true)
			return
		}
		if err != nil {
			resource.Logger.Debug("读取服务器消息失败",
				zap.Int("type", messageType),
				zap.String("serverUrl", conn.RemoteAddr().String()),
				zap.Error(err))
			SetConnectErrorFlag(true)
			return
		}
		if !readHandler(message) {
			return
		}
	}
}

func WriteServerMessage(conn *websocket.Conn) {
	AddMessageListener(func(messageContainer MessageContainer) bool {
		messageType := messageContainer.Type
		data := messageContainer.Data
		err := conn.WriteMessage(messageType, data)
		if err != nil {
			resource.Logger.Info("传递信息失败",
				zap.String("serverUrl", conn.RemoteAddr().String()), zap.Error(err))
			conn.Close()
			return false
		}
		return true
	})
}

func ReConnectServer(ctx context.Context) {
	ticker := time.NewTicker(reConnectPeriod)
	for {
		select {
		case <-ticker.C:
			if connectErrorFlag {
				resource.Logger.Info("尝试重试连接服务器")
				SetConnectErrorFlag(false)
				InitConnectServer(ctx)
			}
		case <-ctx.Done():
			resource.Logger.Info("退出程序")
			return
		}
	}
}

func SetConnectErrorFlag(flag bool) {
	connectErrorFlag = flag
	SysTrayConnectStatusCh <- flag
}

func CloseServer(conn *websocket.Conn) {
	for {
		select {
		case <-CloseCh:
			err := conn.Close()
			if err != nil {
				resource.Logger.Error("关闭服务器连接失败", zap.Error(err))
				return
			}
		}
	}
}

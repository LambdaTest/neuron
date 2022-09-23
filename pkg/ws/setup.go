package ws

import (
	"time"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// RegisterRoutes registers websockets routes
func RegisterRoutes(
	router *gin.RouterGroup,
	logger lumber.Logger,
	synapseManager core.SynapseClientManager,
	synapsePool core.SynapsePoolManager,
) {
	router.GET("/", wsHandleConnect(logger, synapseManager, synapsePool))
}

func wsHandleConnect(
	logger lumber.Logger,
	synapseManager core.SynapseClientManager,
	synapsePool core.SynapsePoolManager,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// upgrade connection
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.Errorf("error upgrading connection %v", err)
			return
		}
		defer ws.Close()
		handleIncomingMessages(ws, logger, synapseManager, synapsePool)
	}
}

func handleIncomingMessages(
	ws *websocket.Conn,
	logger lumber.Logger,
	synapseManager core.SynapseClientManager,
	synapsePool core.SynapsePoolManager) {
	abortConnection := make(chan struct{}, 1)
	for {
		select {
		case <-abortConnection:
			return
		default:
			_, p, err := ws.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure) {
					logger.Infof("websocket closed abnormally")
					return
				}
				logger.Errorf("timeout while trying to read from socket")
				cm := websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, "")
				if closeErr := ws.WriteControl(websocket.CloseMessage, cm, time.Time{}); closeErr != nil {
					logger.Errorf("unable to send close message %v", err)
				}
				logger.Errorf("error reading msg from socket: %v", err)
				return
			}
			msg := string(p)
			go processIncomingMessage(ws, msg, logger, synapseManager, synapsePool, abortConnection)
		}
	}
}

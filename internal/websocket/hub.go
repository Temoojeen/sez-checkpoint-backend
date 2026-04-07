package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketMessage структура сообщения
type WebSocketMessage struct {
	Type             string `json:"type"`                       // "plate_detected", "access_result"
	PlateNumber      string `json:"plateNumber"`                // Номер автомобиля
	AccessGranted    bool   `json:"accessGranted"`              // Доступ разрешен/запрещен
	OrganizationName string `json:"organizationName,omitempty"` // Название организации
	ListName         string `json:"listName,omitempty"`         // Название списка
	ListColor        string `json:"listColor,omitempty"`        // Цвет списка
	Message          string `json:"message,omitempty"`          // Сообщение
	Timestamp        string `json:"timestamp"`                  // Время события
}

// Client представляет WebSocket клиента
type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan []byte
}

// Hub управляет всеми WebSocket соединениями
type Hub struct {
	clients    map[string]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub создает новый хаб
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run запускает хаб
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()
			log.Printf("✅ WebSocket клиент подключен: %s, всего клиентов: %d", client.ID, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
				log.Printf("❌ WebSocket клиент отключен: %s, осталось клиентов: %d", client.ID, len(h.clients))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client.ID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// SendToAll отправляет сообщение всем клиентам
func (h *Hub) SendToAll(message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Ошибка маршалинга сообщения: %v", err)
		return
	}
	h.broadcast <- data
}

// SendPlateEvent отправляет событие о распознанном номере
func (h *Hub) SendPlateEvent(plateNumber string, accessGranted bool, organizationName, listName, listColor, message string) {
	event := WebSocketMessage{
		Type:             "plate_detected",
		PlateNumber:      plateNumber,
		AccessGranted:    accessGranted,
		OrganizationName: organizationName,
		ListName:         listName,
		ListColor:        listColor,
		Message:          message,
		Timestamp:        time.Now().Format("2006-01-02 15:04:05"),
	}
	h.SendToAll(event)
	log.Printf("📡 WebSocket событие отправлено: номер %s, доступ %v", plateNumber, accessGranted)
}

// ServeWebSocket обрабатывает WebSocket подключения
func (h *Hub) ServeWebSocket(c *gin.Context) {
	// Обновляем соединение до WebSocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // В продакшене нужно настроить правильно
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("❌ Ошибка обновления до WebSocket: %v", err)
		return
	}

	// Создаем клиента
	clientID := c.Query("clientId")
	if clientID == "" {
		clientID = c.ClientIP() + "_" + time.Now().Format("20060102150405")
	}

	client := &Client{
		ID:   clientID,
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	// Регистрируем клиента
	h.register <- client

	// Запускаем горутину для отправки сообщений
	go h.writePump(client)

	// Запускаем горутину для чтения сообщений
	go h.readPump(client)
}

// writePump отправляет сообщения клиенту
func (h *Hub) writePump(client *Client) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
		h.unregister <- client
	}()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump читает сообщения от клиента
func (h *Hub) readPump(client *Client) {
	defer func() {
		client.Conn.Close()
		h.unregister <- client
	}()

	client.Conn.SetReadLimit(512)
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ WebSocket ошибка: %v", err)
			}
			break
		}
	}
}

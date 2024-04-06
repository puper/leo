package uniqid

type EventType string

const (
	TypeServerIdUpdate  EventType = "serverIdUpdate"
	TypeServerIdsUpdate EventType = "serverIdsUpdate"
	TypeError           EventType = "error"
)

type Event struct {
	Type  EventType
	Error error
}

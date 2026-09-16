package protocol

type TerminalOpenPayload struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

type TerminalInputPayload struct {
	Data string `json:"data"`
}

type TerminalResizePayload struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

type TerminalClosePayload struct {
	Reason string `json:"reason,omitempty"`
}

type TerminalOpenedPayload struct {
	SessionID string `json:"session_id"`
}

type TerminalExitPayload struct {
	Code int `json:"code"`
}

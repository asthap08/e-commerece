package audit

type AuditLog struct {
	ID       string `json:"id"`
	ActorID  string `json:"actor_id"`
	Action   string `json:"action"`
	Entity   string `json:"entity"`
	EntityID string `json:"entity_id"`
}

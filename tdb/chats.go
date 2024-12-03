package tdb

const (
	getTranscribeChat    = `SELECT enabled FROM transcribe_chats WHERE chat_id = $1`
	createTranscribeChat = `INSERT INTO transcribe_chats (enabled, chat_id) VALUES ($1, $2)`
	updateTranscribeChat = `UPDATE transcribe_chats SET enabled = $1 WHERE chat_id = $2`
)

func (t *TranscriptionDB) GetTranscribeChat(chatId string) (enabled bool, err error) {
	err = t.db.QueryRow(getTranscribeChat, chatId).Scan(&enabled)
	return
}

func (t *TranscriptionDB) CreateTranscribeChat(chatId string, enabled bool) error {
	_, err := t.db.Exec(createTranscribeChat, enabled, chatId)
	return err
}

func (t *TranscriptionDB) UpdateChatEnabled(chatId string, enabled bool) error {
	_, err := t.db.Exec(updateTranscribeChat, enabled, chatId)
	return err
}

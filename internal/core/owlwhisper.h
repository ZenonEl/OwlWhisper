#ifndef OWLWHISPER_H
#define OWLWHISPER_H

#ifdef __cplusplus
extern "C" {
#endif

// Инициализация и управление
int StartOwlWhisper();
int StopOwlWhisper();

// Отправка сообщений
int SendMessage(char* text);
int SendMessageToPeer(char* peerID, char* text);

// Получение информации
char* GetMyPeerID();
char* GetPeers();
char* GetConnectionStatus();

// Получение истории
char* GetChatHistory(char* peerID);
char* GetChatHistoryLimit(char* peerID, int limit);

// Подключение к пирам
int ConnectToPeer(char* peerID);

// Освобождение памяти
void FreeString(char* str);

#ifdef __cplusplus
}
#endif

#endif // OWLWHISPER_H 
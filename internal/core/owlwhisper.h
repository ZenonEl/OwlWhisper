#ifndef OWLWHISPER_H
#define OWLWHISPER_H

#ifdef __cplusplus
extern "C" {
#endif

// Инициализация и управление
int StartOwlWhisper();
int StartOwlWhisperWithKey(char* keyBytes, int keyLength);
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

// Функции для работы с профилями
extern char* GetMyProfile();
extern int UpdateMyProfile(char* nickname);
extern char* GetPeerProfile(char* peer_id);

// Функции для настройки логирования
extern int SetLogLevel(int level);
extern int SetLogOutput(int output, char* log_dir);

// Функции для управления памятью
extern void FreeString(char* str);

#ifdef __cplusplus
}
#endif

#endif // OWLWHISPER_H 
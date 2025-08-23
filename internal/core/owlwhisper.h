#ifndef OWLWHISPER_H
#define OWLWHISPER_H

#ifdef __cplusplus
extern "C" {
#endif

// Инициализация и управление
int StartOwlWhisper();
int StartOwlWhisperWithKey(char* keyBytes, int keyLength);
int StopOwlWhisper();

// Генерация ключей
char* GenerateNewKeyPair();
char* GenerateNewKeyBytes();

// Отправка сообщений
int SendMessage(char* text);
int SendMessageToPeer(char* peerID, char* text);

// Получение информации
char* GetMyPeerID();
char* GetConnectedPeers();
char* GetProtectedPeers();
char* GetConnectionStatus();

// Управление защищенными пирами
int AddProtectedPeer(char* peerID);
int RemoveProtectedPeer(char* peerID);
int IsProtectedPeer(char* peerID);
char* GetConnectionLimits();

// Автопереподключение к защищенным пирам
int EnableAutoReconnect();
int DisableAutoReconnect();
int IsAutoReconnectEnabled();
int GetReconnectAttempts(char* peerID);

// Получение истории
char* GetChatHistory(char* peerID);
char* GetChatHistoryLimit(char* peerID, int limit);

// Подключение к пирам
int ConnectToPeer(char* peerID);

// Поиск и диагностика
char* FindPeer(char* peerID);
char* FindProvidersForContent(char* contentID);
char* GetNetworkStats();
char* GetConnectionQuality(char* peerID);

// Анонсирование контента
int ProvideContent(char* contentID);


// Функции для настройки логирования
extern int SetLogLevel(int level);
extern int SetLogOutput(int output, char* log_dir);

// Функции для управления памятью
extern void FreeString(char* str);

#ifdef __cplusplus
}
#endif

#endif // OWLWHISPER_H 
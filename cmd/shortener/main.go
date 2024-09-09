package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
)

// Структура для хранения пар URL
type urlPair struct {
	Original string `json:"original"`
	Short    string `json:"short"`
}

// Хранилище URL с мьютексом для потокобезопасности
type urlStore struct {
	urls map[string]urlPair
	mu   sync.Mutex
}

// Функция конструктор для создания нового хранилища пар URL
func newURLStore() *urlStore {
	return &urlStore{
		urls: make(map[string]urlPair),
	}
}

// Регистрируем обработчик
func webhook(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request: %s %s", r.Method, r.URL.Path)
	switch r.Method {
	case http.MethodPost:
		handlePOST(w, r)
	case http.MethodGet:
		handleGET(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

}

/* запрос для терминала командной строки: curl -v -X POST  "http://localhost:8080" -H "Content-Type: text/plain" -H "Host: localhost:8080" -d
"https://practicum.yandex.ru/" */
// Обработчик POST-запросов для создания нового короткого URL
func handlePOST(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/" {
		http.Error(w, "Invalid request method or path", http.StatusBadRequest)
		return
	}
	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	originalURL := string(body)
	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	// Создаем новый короткий URL
	shortURL, err := store.add(string(body))
	if err != nil {
		http.Error(w, "Failed to create short URL", http.StatusInternalServerError)
		return
	}
	// Отправляем ответ с кодом 201 и коротким URL
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("content-type", "text/plain")
	w.Write([]byte(shortURL))
}

// Метод добавления новой пары URL в хранилище
func (s *urlStore) add(original string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Генерируем короткий идентификатор
	short := generateShortID(8)
	// Добавляем пару URL в хранилище
	s.urls[short] = urlPair{
		Original: original,
		Short:    short,
	}
	// Возвращаем строку короткого URL
	return fmt.Sprintf("http://localhost:8080/%s", short), nil
}

// Функция генерации случайного короткого идентификатора заданной длины
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateShortID(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

/* запрос для терминала командной строки: curl -v GET "http://localhost:8080/YxL8Q2B1" -H 'Host: localhost:8080' -H 'Content-Type: text/plain' */
// Обработчик GET-запросов для перенаправления на оригинальный URL
func handleGET(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || len(r.URL.Path) <= 1 {
		http.Error(w, "Invalid request method or path", http.StatusBadRequest)
		return
	}
	// Извлекаем короткий идентификатор из пути
	short := r.URL.Path[1:]

	// Получаем оригинальный URL по короткому идентификатору
	original, exists := store.get(short)
	if !exists {
		http.Error(w, "Short URL not found", http.StatusBadRequest)
		return
	}
	// Отправляем ответ с кодом 307 и оригинальным URL
	w.Header().Set("Location", original)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// Метод получения оригинального URL по короткому идентификатору
func (s *urlStore) get(short string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pair, exists := s.urls[short]
	if !exists {
		return "", false
	}
	return pair.Original, true
}

var store *urlStore

// Функция main для запуска сервера
func main() {
	mux := http.NewServeMux()
	log.Println("Server started")

	store = newURLStore()

	mux.HandleFunc("/", webhook)
	log.Fatal(http.ListenAndServe(":8080", mux))
}

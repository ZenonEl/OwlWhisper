package main

import _ "OwlWhisper/internal/core"

// Этот файл нужен только для компиляции shared library
// Все функции экспортируются через CGo из core пакета
func main() {
	// Пустая main функция
} 
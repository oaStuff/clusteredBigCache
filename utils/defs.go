package utils

//AppLogger is an interface defining logging for clusteredBigCache
type AppLogger interface {
	Info(msg string)
	Warn(msg string)
	Critical(msg string)
	Error(msg string)
}

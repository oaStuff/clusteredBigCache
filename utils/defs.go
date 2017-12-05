package utils


type AppLogger interface {
	Info(msg string)
	Warn(msg string)
	Critical(msg string)
	Error(msg string)
}



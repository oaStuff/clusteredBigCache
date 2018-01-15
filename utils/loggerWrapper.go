package utils

//Info uses logger to print info messages
func Info(logger AppLogger, msg string) {
	if nil != logger {
		logger.Info(msg)
	}
}

//Error uses logger to print error messages
func Error(logger AppLogger, msg string) {
	if nil != logger {
		logger.Error(msg)
	}
}

//Warn uses logger to print warning messages
func Warn(logger AppLogger, msg string) {
	if nil != logger {
		logger.Warn(msg)
	}
}

//Critical uses logger to print critical messages
func Critical(logger AppLogger, msg string) {
	if nil != logger {
		logger.Critical(msg)
	}
}

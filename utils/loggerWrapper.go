package utils


func Info(logger AppLogger, msg string)  {
	if nil != logger {
		logger.Info(msg)
	}
}

func Error(logger AppLogger, msg string)  {
	if nil != logger {
		logger.Error(msg)
	}
}


func Warn(logger AppLogger, msg string)  {
	if nil != logger {
		logger.Warn(msg)
	}
}


func Critical(logger AppLogger, msg string)  {
	if nil != logger {
		logger.Critical(msg)
	}
}





package utils

import (
	"gopkg.in/ini.v1"
)

type IniParser struct {
	confReader *ini.File // config reader
}

type IniParserError struct {
	errorInfo string
}

func (e *IniParserError) Error() string { return e.errorInfo }

func (parser *IniParser) Load(configFileName string) error {
	conf, err := ini.Load(configFileName)
	if err != nil {
		parser.confReader = nil
		return err
	}
	parser.confReader = conf
	return nil
}

func (parser *IniParser) GetString(section string, key string) string {
	if parser.confReader == nil {
		return ""
	}

	s := parser.confReader.Section(section)
	if s == nil {
		return ""
	}

	return s.Key(key).String()
}

func (parser *IniParser) GetInt32(section string, key string) int32 {
	if parser.confReader == nil {
		return 0
	}

	s := parser.confReader.Section(section)
	if s == nil {
		return 0
	}

	valueInt, _ := s.Key(key).Int()

	return int32(valueInt)
}

func (parser *IniParser) GetUint32(section string, key string) uint32 {
	if parser.confReader == nil {
		return 0
	}

	s := parser.confReader.Section(section)
	if s == nil {
		return 0
	}

	valueInt, _ := s.Key(key).Uint()

	return uint32(valueInt)
}

func (parser *IniParser) GetInt64(section string, key string) int64 {
	if parser.confReader == nil {
		return 0
	}

	s := parser.confReader.Section(section)
	if s == nil {
		return 0
	}

	valueInt, _ := s.Key(key).Int64()
	return valueInt
}

func (parser *IniParser) GetUint64(section string, key string) uint64 {
	if parser.confReader == nil {
		return 0
	}

	s := parser.confReader.Section(section)
	if s == nil {
		return 0
	}

	valueInt, _ := s.Key(key).Uint64()
	return valueInt
}

func (parser *IniParser) GetFloat32(section string, key string) float32 {
	if parser.confReader == nil {
		return 0
	}

	s := parser.confReader.Section(section)
	if s == nil {
		return 0
	}

	valueFloat, _ := s.Key(key).Float64()
	return float32(valueFloat)
}

func (parser *IniParser) GetFloat64(section string, key string) float64 {
	if parser.confReader == nil {
		return 0
	}

	s := parser.confReader.Section(section)
	if s == nil {
		return 0
	}

	valueFloat, _ := s.Key(key).Float64()
	return valueFloat
}

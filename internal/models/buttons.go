package models

var (
	MainButtons = []ButtonOption{
		{Value: "Schedule", Display: "Расписание"},
		{Value: "PrevSchedule", Display: "Предыдущее"},
		{Value: "NextSchedule", Display: "Следующее"},
		{Value: "Settings", Display: "Настройки"},
		{Value: "Support", Display: "Техподдержка"},
	}
	RoleButtons = []ButtonOption{
		{Value: "Student", Display: "Студент"},
		{Value: "Teacher", Display: "Преподаватель"},
	}
	TypeButtons = []ButtonOption{
		{Value: "OIT", Display: "ОИТ"},
		{Value: "OAR", Display: "ОАР"},
		{Value: "SO", Display: "СО"},
		{Value: "MMO", Display: "ММО"},
		{Value: "OEP", Display: "ОЭП"},
	}
	GroupOITButtons = []ButtonOption{
		{Value: "IS1_11", Display: "ИС1-11"},
		{Value: "IS1_13", Display: "ИС1-13"},
		{Value: "IS1_15", Display: "ИС1-15"},
		{Value: "IS1_21", Display: "ИС1-21"},
		{Value: "IS1_23", Display: "ИС1-23"},
		{Value: "IS1_25", Display: "ИС1-25"},
		{Value: "IS1_31", Display: "ИС1-31"},
		{Value: "IS1_33", Display: "ИС1-33"},
		{Value: "IS1_35", Display: "ИС1-35"},
		{Value: "IS1_41", Display: "ИС1-41"},
		{Value: "IS1_43", Display: "ИС1-43"},
		{Value: "IS1_45", Display: "ИС1-45"},
		{Value: "SA1_11", Display: "СА1-11"},
		{Value: "SA1_21", Display: "СА1-21"},
		{Value: "SA1_31", Display: "СА1-31"},
		{Value: "SA1_41", Display: "СА1-41"},
		{Value: "IB1_11", Display: "ИБ1-11"},
		{Value: "IB1_21", Display: "ИБ1-21"},
		{Value: "IB1_31", Display: "ИБ1-31"},
		{Value: "IB1_41", Display: "ИИ1-41"},
	}
	GroupOARButtons = []ButtonOption{}
	GroupSOButtons  = []ButtonOption{}
	GroupMMOButtons = []ButtonOption{}
	GroupOEPButtons = []ButtonOption{}
)

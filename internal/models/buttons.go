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
		{Value: "IS1_11", Display: "ОИТ"},
		{Value: "IS1_13", Display: "ОАР"},
		{Value: "IS1_15", Display: "СО"},
		{Value: "IS1_21", Display: "ММО"},
		{Value: "OEP", Display: "ОЭП"},
		{Value: "OIT", Display: "ОИТ"},
		{Value: "OAR", Display: "ОАР"},
		{Value: "SO", Display: "СО"},
		{Value: "MMO", Display: "ММО"},
		{Value: "OEP", Display: "ОЭП"},
		{Value: "OIT", Display: "ОИТ"},
		{Value: "OAR", Display: "ОАР"},
		{Value: "SO", Display: "СО"},
		{Value: "MMO", Display: "ММО"},
		{Value: "OEP", Display: "ОЭП"},
	}
	GrosupOITButtons = []string{"IS1_11", "IS1_13", "IS1_15", "IS1_21",
		"IS1_23", "IS1_25", "IS1_31", "IS1_33",
		"IS1_35", "IS1_41", "IS1_43", "IS1_45",
		"SA1_11", "SA1_21", "SA1_31", "SA1_41",
		"IB1_11", "IB1_21", "IB1_41", "IB1_21"}
	GroupOARButtons = []string{}
	GroupSOButtons  = []string{}
	GroupMMOButtons = []string{}
	GroupOEPButtons = []string{}
)

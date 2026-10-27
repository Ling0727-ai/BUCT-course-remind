package utils

// Defaults 封装所有固定返回值和默认值的结构体
// 通过修改此结构体可集中控制所有固定返回值
var Defaults = struct {
	// HTTP 状态码
	Status struct {
		OK                  int
		Created             int
		BadRequest          int
		Unauthorized        int
		Forbidden           int
		NotFound            int
		Conflict            int
		InternalServerError int
		ServiceUnavailable  int
		TooManyRequests     int
	}

	// 认证相关消息
	Auth struct {
		NotLoggedIn          string
		RequestFormatError   string
		InvalidCredentials   string
		LoginSuccess         string
		RegisterSuccess      string
		LogoutSuccess        string
		RequireAdmin         string
		ResetPasswordSuccess string
		StudentInfoSuccess   string
		LoginRateLimited     string
	}

	// 用户相关消息
	User struct {
		NotFound            string
		AlreadyExists       string
		UpdateFailed        string
		PasswordMismatch    string
		PasswordTooShort    string
		PasswordProcessFail string
		EmailAndPwdRequired string
	}

	// 邮箱相关消息
	Email struct {
		Empty         string
		InvalidFormat string
		AlreadyUsed   string
		NotFound      string
		SendFailed    string
		UpdateSuccess string
	}

	// 数据操作相关消息
	Data struct {
		GetFailed     string
		CreateFailed  string
		UpdateFailed  string
		DeleteFailed  string
		RestoreFailed string
		ClearFailed   string
		QueryFailed   string
		ParseFailed   string
		NotFound      string
		AlreadyExists string
		NotAllowed    string
	}

	// 作业/任务相关消息
	Assignment struct {
		CompleteSuccess    string
		UncompleteSuccess  string
		DeleteSuccess      string
		RestoreSuccess     string
		PermDeleteSuccess  string
		ClearSuccess       string
		AlreadyCompleted   string
		AlreadyDeleted     string
		GetFailed          string
		GetStatsFailed     string
		GetCompletedFailed string
		GetDeletedFailed   string
		QueryFailed        string
		NotFound           string
		RemindSuccess      string
		CreateReminderFail string
	}

	// 待办事项相关消息
	Todo struct {
		TitleEmpty           string
		HoursInvalid         string
		DueDateInvalid       string
		CompleteSuccess      string
		UncompleteSuccess    string
		DeleteSuccess        string
		RestoreSuccess       string
		PermDeleteSuccess    string
		ClearSuccess         string
		RetentionHint        string
		CreateSuccess        string
		UpdateSuccess        string
		NotFound             string
		GetFailed            string
		GetDeletedFailed     string
		GetStatsFailed       string
		AlreadyCompletedHint string
		AlreadyDeletedHint   string
		NoEmailHint          string
		RemindSuccess        string
	}

	// 验证码相关消息
	Verification struct {
		Empty            string
		InvalidOrExpired string
		SendFailed       string
		SendSuccess      string
		VerifySuccess    string
		VerifyFailed     string
		RateLimited      string
		ServiceError     string
		TestModeNote     string
	}

	// 提醒相关消息
	Reminder struct {
		SendSuccess      string
		CreateSuccess    string
		DeleteSuccess    string
		GetFailed        string
		TimeInvalid      string
		TimePast         string
		MissingID        string
		TestEmailSuccess string
	}

	// 黑名单相关消息
	Blacklist struct {
		GetSuccess     string
		AddSuccess     string
		RemoveSuccess  string
		ClearSuccess   string
		SubjectIDEmpty string
	}

	// 设置相关消息
	Settings struct {
		GetFailed        string
		SaveFailed       string
		SaveSuccess      string
		DBConnectFail    string
		EmailSaveSuccess string
	}

	// 加密相关消息
	Crypto struct {
		DecryptFailed     string
		EncryptFailed     string
		RequireEncryption string
	}

	// 数据库相关消息
	Database struct {
		ConnectFailed string
		QueryFailed   string
	}

	// 其他通用消息
	Misc struct {
		OK              string
		Pong            string
		Success         string
		Fail            string
		Unauthorized    string
		ForBidden       string
		RequestError    string
		ProcessError    string
		NoUpdatedFields string
	}
}{
	Status: struct {
		OK                  int
		Created             int
		BadRequest          int
		Unauthorized        int
		Forbidden           int
		NotFound            int
		Conflict            int
		InternalServerError int
		ServiceUnavailable  int
		TooManyRequests     int
	}{
		OK:                  200,
		Created:             201,
		BadRequest:          400,
		Unauthorized:        401,
		Forbidden:           403,
		NotFound:            404,
		Conflict:            409,
		InternalServerError: 500,
		ServiceUnavailable:  503,
		TooManyRequests:     429,
	},

	Auth: struct {
		NotLoggedIn          string
		RequestFormatError   string
		InvalidCredentials   string
		LoginSuccess         string
		RegisterSuccess      string
		LogoutSuccess        string
		RequireAdmin         string
		ResetPasswordSuccess string
		StudentInfoSuccess   string
		LoginRateLimited     string
	}{
		NotLoggedIn:          "未登录",
		RequestFormatError:   "请求格式错误",
		InvalidCredentials:   "用户名/邮箱和密码不能为空",
		LoginSuccess:         "登录成功",
		RegisterSuccess:      "注册成功",
		LogoutSuccess:        "登出成功",
		RequireAdmin:         "需要管理员权限",
		ResetPasswordSuccess: "密码重置成功",
		StudentInfoSuccess:   "学生信息更新成功",
		LoginRateLimited:     "登录尝试过于频繁，请稍后再试",
	},

	User: struct {
		NotFound            string
		AlreadyExists       string
		UpdateFailed        string
		PasswordMismatch    string
		PasswordTooShort    string
		PasswordProcessFail string
		EmailAndPwdRequired string
	}{
		NotFound:            "用户不存在",
		AlreadyExists:       "用户已存在",
		UpdateFailed:        "更新失败，请重试",
		PasswordMismatch:    "密码不匹配",
		PasswordTooShort:    "密码长度至少6位",
		PasswordProcessFail: "密码处理失败",
		EmailAndPwdRequired: "邮箱和新密码不能为空",
	},

	Email: struct {
		Empty         string
		InvalidFormat string
		AlreadyUsed   string
		NotFound      string
		SendFailed    string
		UpdateSuccess string
	}{
		Empty:         "邮箱地址不能为空",
		InvalidFormat: "邮箱格式不正确",
		AlreadyUsed:   "该邮箱已被其他用户使用",
		NotFound:      "该邮箱未注册",
		SendFailed:    "邮件发送失败",
		UpdateSuccess: "邮箱修改成功",
	},

	Data: struct {
		GetFailed     string
		CreateFailed  string
		UpdateFailed  string
		DeleteFailed  string
		RestoreFailed string
		ClearFailed   string
		QueryFailed   string
		ParseFailed   string
		NotFound      string
		AlreadyExists string
		NotAllowed    string
	}{
		GetFailed:     "获取失败",
		CreateFailed:  "创建失败",
		UpdateFailed:  "更新失败",
		DeleteFailed:  "删除失败",
		RestoreFailed: "恢复失败",
		ClearFailed:   "清空失败",
		QueryFailed:   "查询失败",
		ParseFailed:   "解析数据失败",
		NotFound:      "不存在",
		AlreadyExists: "已存在",
		NotAllowed:    "不允许操作",
	},

	Assignment: struct {
		CompleteSuccess    string
		UncompleteSuccess  string
		DeleteSuccess      string
		RestoreSuccess     string
		PermDeleteSuccess  string
		ClearSuccess       string
		AlreadyCompleted   string
		AlreadyDeleted     string
		GetFailed          string
		GetStatsFailed     string
		GetCompletedFailed string
		GetDeletedFailed   string
		QueryFailed        string
		NotFound           string
		RemindSuccess      string
		CreateReminderFail string
	}{
		CompleteSuccess:    "作业已标记为完成",
		UncompleteSuccess:  "已撤销完成状态",
		DeleteSuccess:      "作业已移至回收站",
		RestoreSuccess:     "作业已恢复",
		PermDeleteSuccess:  "作业已永久删除",
		ClearSuccess:       "已清空回收站",
		AlreadyCompleted:   "该作业已完成，无需提醒",
		AlreadyDeleted:     "该作业已删除，无法提醒",
		GetFailed:          "获取作业列表失败",
		GetStatsFailed:     "获取统计失败",
		GetCompletedFailed: "获取已完成作业失败",
		GetDeletedFailed:   "获取已删除作业失败",
		QueryFailed:        "查询作业失败",
		NotFound:           "作业不存在",
		RemindSuccess:      "提醒已立即发送：",
		CreateReminderFail: "创建提醒失败",
	},

	Todo: struct {
		TitleEmpty           string
		HoursInvalid         string
		DueDateInvalid       string
		CompleteSuccess      string
		UncompleteSuccess    string
		DeleteSuccess        string
		RestoreSuccess       string
		PermDeleteSuccess    string
		ClearSuccess         string
		RetentionHint        string
		CreateSuccess        string
		UpdateSuccess        string
		NotFound             string
		GetFailed            string
		GetDeletedFailed     string
		GetStatsFailed       string
		AlreadyCompletedHint string
		AlreadyDeletedHint   string
		NoEmailHint          string
		RemindSuccess        string
	}{
		TitleEmpty:           "待办标题不能为空",
		HoursInvalid:         "小时数必须大于0",
		DueDateInvalid:       "截止日期格式错误",
		CompleteSuccess:      "待办事项已完成",
		UncompleteSuccess:    "待办事项完成状态已撤销",
		DeleteSuccess:        "待办事项已移至回收站",
		RestoreSuccess:       "待办事项已恢复",
		PermDeleteSuccess:    "待办事项已永久删除",
		ClearSuccess:         "已清空回收站",
		RetentionHint:        "，将保留直到删除",
		CreateSuccess:        "待办事项创建成功",
		UpdateSuccess:        "待办事项更新成功",
		NotFound:             "待办事项不存在",
		GetFailed:            "获取待办列表失败",
		GetDeletedFailed:     "获取已删除待办失败",
		GetStatsFailed:       "获取统计失败",
		AlreadyCompletedHint: "已完成，无需提醒",
		AlreadyDeletedHint:   "该待办事项已删除，无法提醒",
		NoEmailHint:          "未找到用户邮箱，请在设置中配置邮箱",
		RemindSuccess:        "提醒已发送: ",
	},

	Verification: struct {
		Empty            string
		InvalidOrExpired string
		SendFailed       string
		SendSuccess      string
		VerifySuccess    string
		VerifyFailed     string
		RateLimited      string
		ServiceError     string
		TestModeNote     string
	}{
		Empty:            "邮箱和验证码不能为空",
		InvalidOrExpired: "验证码无效或已过期",
		SendFailed:       "验证码发送失败，请稍后重试",
		SendSuccess:      "验证码已发送到您的邮箱",
		VerifySuccess:    "验证成功",
		VerifyFailed:     "验证失败，请重试",
		RateLimited:      "请等待 1 分钟后再次发送验证码",
		ServiceError:     "服务异常，请重试",
		TestModeNote:     "邮件服务暂时不可用，使用测试模式",
	},

	Reminder: struct {
		SendSuccess      string
		CreateSuccess    string
		DeleteSuccess    string
		GetFailed        string
		TimeInvalid      string
		TimePast         string
		MissingID        string
		TestEmailSuccess string
	}{
		SendSuccess:      "提醒邮件已发送",
		CreateSuccess:    "创建提醒失败",
		DeleteSuccess:    "提醒已删除",
		GetFailed:        "获取提醒失败",
		TimeInvalid:      "时间格式不正确，请使用 2006-01-02 15:04:05",
		TimePast:         "提醒时间不能早于当前时间",
		MissingID:        "缺少提醒 ID",
		TestEmailSuccess: "测试邮件已发送到: ",
	},

	Blacklist: struct {
		GetSuccess     string
		AddSuccess     string
		RemoveSuccess  string
		ClearSuccess   string
		SubjectIDEmpty string
	}{
		GetSuccess:     "获取黑名单失败",
		AddSuccess:     "已加入黑名单",
		RemoveSuccess:  "已从黑名单移除",
		ClearSuccess:   "黑名单已清空",
		SubjectIDEmpty: "subject_id 不能为空",
	},

	Settings: struct {
		GetFailed        string
		SaveFailed       string
		SaveSuccess      string
		DBConnectFail    string
		EmailSaveSuccess string
	}{
		GetFailed:        "获取设置失败",
		SaveFailed:       "保存失败",
		SaveSuccess:      "设置保存成功",
		DBConnectFail:    "数据库连接失败",
		EmailSaveSuccess: "邮箱设置保存成功",
	},

	Crypto: struct {
		DecryptFailed     string
		EncryptFailed     string
		RequireEncryption string
	}{
		DecryptFailed:     "数据解密失败",
		EncryptFailed:     "数据加密失败",
		RequireEncryption: "必须使用加密传输",
	},

	Database: struct {
		ConnectFailed string
		QueryFailed   string
	}{
		ConnectFailed: "数据库连接失败",
		QueryFailed:   "数据库查询失败",
	},

	Misc: struct {
		OK              string
		Pong            string
		Success         string
		Fail            string
		Unauthorized    string
		ForBidden       string
		RequestError    string
		ProcessError    string
		NoUpdatedFields string
	}{
		OK:              "ok",
		Pong:            "pong",
		Success:         "成功",
		Fail:            "失败",
		Unauthorized:    "未授权",
		ForBidden:       "禁止访问",
		RequestError:    "请求参数错误",
		ProcessError:    "处理失败",
		NoUpdatedFields: "无可更新的字段",
	},
}

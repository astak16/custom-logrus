基于 `logrus` 实现一个日志输出的工具，具备的功能：

1. 自定义输出内容
2. 输出到文件
3. 输出到控制台

自定义输出内容，通过 `logrus.setFormatter` 方法设置自定义的 `Formatter`，实现自定义输出内容的功能

`Formatter` 接口只有一个方法 `Format`，接收一个 `Entry` 类型的参数，返回一个 `byte` 类型的切片和一个 `error` 类型的错误。`Entry` 类型是 `logrus` 的日志实体，包含了日志的级别、时间、消息等信息

```go
type Formatter interface {
  Format(*Entry) ([]byte, error)
}
```

输出到控制台和输出到文件，需要分别实现两个 `LogFileFormatter` 和 `LogConsoleFormatter`

## 按日期分割

按日期分类，就是根据日期将日志分别输出到不同的文件中，例如 `2024-06-06.log`、`2024-06-07.log` 等，如果需要按照时分秒分割，自己格式化时间即可

### LogFileFormatter

定义一个 `LogFileFormatter` 结构体，实现 `Formatter` 接口的 `Format` 方法，就可以实现自定义输出到文件内容的功能

具体内容如下：

1. `entry.Caller` 是一个指向 `runtime.Frame` 结构体的指针，它包含了调用日志记录函数的代码文件和行号信息
   - `file = filepath.Base(entry.Caller.File)` 从 `entry.Caller.File` 中提取文件名，例如 `"main.go"`
   - `len = entry.Caller.Line` 获取调用日志记录函数的代码行号
2. `entry.Message` 是一个 `string` 类型的消息内容
3. 最终日志的内容，包括以下几个部分:
   - `[%s]`：日志级别，如 `"INFO"`、`"ERROR"` 等，使用 `strings.ToUpper(entry.Level.String())` 将其转换为大写
   - `%s`：时间戳字符串
   - `[%s:%d]`：文件名和行号信息，如果 `entry.Caller` 为空，则这部分为空
   - `%s`：日志消息内容，即 `entry.Message`

```go
type LogFileFormatter struct{}

func (s *LogFileFormatter) Format(entry *logrus.Entry) ([]byte, error) {
  timestamp := time.Now().Local().Format("2006-01-02 15:04:05")
  var file string
  var len int
  if entry.Caller != nil {
    // 提取文件名
    file = filepath.Base(entry.Caller.File)
    // 获取调用日志记录函数的代码行号
    len = entry.Caller.Line
  }
  msg := fmt.Sprintf("[%s] %s [%s:%d] %s\n", strings.ToUpper(entry.Level.String()), timestamp, file, len, entry.Message)
  return []byte(msg), nil
}
```

### LogConsoleFormatter

定义一个 `LogConsoleFormatter` 结构体，实现 `Formatter` 接口的 `Format` 方法，就可以实现自定义输出到控制台内容的功能

具体内容如下：

1. 通过 `entry.Level` 获取日志级别，然后根据不同的级别设置不同的颜色
2. 设置缓冲区 `b`，如果 `entry.Buffer` 为空，则创建一个新的 `bytes.Buffer` 对象，否则使用 `entry.Buffer`
3. 将内容写入缓冲区 `b`，包括以下几个部分：
   - `console`：表示日志内容的前缀
   - `\033[3%dm`：设置输出的颜色，`%d` 是一个占位符，根据不同的颜色设置不同的值
   - `entry.Level`：日志级别，如 `"INFO"`、`"ERROR"` 等
   - `timestamp`：时间戳字符串
   - `fileVal`：文件名和行号信息，如果 `entry.Caller` 为空，则这部分为空
   - `entry.Message`：日志消息内容

```go
var (
  ccRed    = 1
  ccYellow = 3
  ccBlue   = 4
  ccCyan   = 6
  ccGray   = 7
)

type LogConsoleFormatter struct{}

func (s *LogConsoleFormatter) Format(entry *logrus.Entry) ([]byte, error) {
  timestamp := time.Now().Local().Format("2006-01-02 15:04:05")
  var color int
  switch entry.Level {
  case logrus.ErrorLevel:
    color = ccRed
  case logrus.WarnLevel:
    color = ccYellow
  case logrus.InfoLevel:
    color = ccBlue
  case logrus.DebugLevel:
    color = ccCyan
  default:
    color = ccGray
  }
  // 设置 buffer 缓冲区
  var b *bytes.Buffer
  if entry.Buffer == nil {
    b = &bytes.Buffer{}
  } else {
    b = entry.Buffer
  }
  fileVal := fmt.Sprintf("%s:%d", filepath.Base(entry.Caller.File), entry.Caller.Line)
  fmt.Fprintf(b, "[%s] \033[3%dm[%s]\033[0m [%s] %s %s\n", "console", color, entry.Level, timestamp, fileVal, entry.Message)
  return b.Bytes(), nil
}
```

### hook

需要实现不同的格式化内容，我们需要借助 `logrus` 的 `Hook` 接口，分别对 `File` 和 `Console` 实现 `Levels` 和 `Fire` 方法

先来实现 `Console` 的 `Hook`，具体内容如下：

- `Levels` 方法返回一个 `logrus.Level` 类型的切片，表示需要处理的日志级别
- `Fire` 方法接收一个设置自定义的 `Formatter` 对象，然后在函数结束时恢复原来的 `Formatter` 对象，最后将日志内容写入到控制台

```go
type ConsoleHook struct {
  formatter logrus.Formatter
}
func (hook *ConsoleHook) Levels() []logrus.Level {
  return logrus.AllLevels
}
func (hook *ConsoleHook) Fire(entry *logrus.Entry) error {
  originalFormatter := entry.Logger.Formatter
  entry.Logger.Formatter = hook.formatter
  defer func() { entry.Logger.Formatter = originalFormatter }()
  line, err := entry.Logger.Formatter.Format(entry)
  if err != nil {
    return err
  }
  _, err = os.Stdout.Write(line)
  return err
}
```

`FileHook` 和 `ConsoleHook` 的实现方式类似，只是 `FileHook` 多了一个 `file` 字段，用来存储日志文件的指针，具体内容如下：

```go
type FileHook struct {
  formatter logrus.Formatter
  file      *os.File
}

func (hook *FileHook) Levels() []logrus.Level {
  return logrus.AllLevels
}

func (hook *FileHook) Fire(entry *logrus.Entry) error {
  originalFormatter := entry.Logger.Formatter
  entry.Logger.Formatter = hook.formatter
  defer func() { entry.Logger.Formatter = originalFormatter }()
  line, err := entry.Logger.Formatter.Format(entry)
  if err != nil {
    return err
  }
  _, err = hook.file.Write(line)
  return err
}
```

### 初始化

我们先按照日期进行分组输出日志，即每分钟一个文件（测试方便，后面可以改成每天一个文件），文件名为 `2024-06-06.log`、`2024-06-07.log` 等

我们定义一个结构体，来初始化做这件事，这个结构体包含以下几个字段：

- `Date`：表示需要拆分的维度，可以按照自定义时间格式拆分
- `Path`：表示日志存储的路径
- `Name`：表示日志的文件的前缀

```go
type DateLogConfig struct {
  Date string
  Path string
  Name string
}
```

准备好 `DateLogConfig` 结构体之后，我们可以定义一个 `NewDateLog` 函数，用来初始化 `DateLogConfig` 结构体

```go
func NewDateLog(d *DateLogConfig) *DateLogConfig {
  return &DateLogConfig{
    Date: d.Date,
    Path: d.Path,
    Name: d.Name,
  }
}
```

然后在定义一个 `init` 方法，用来完成日志文件的初始化工作，具体内容如下：

1. 实例化 `logrus` 对象
2. 设置是否输出文件名和行号信息
3. 将 `logrus` 的默认输出丢弃，确保日志只通过 `hooks` 输出
4. 添加控制台输出的 `hook`
5. 添加文件输出的 `hook`
6. 将 `logrus` 对象返回出去
   - 外面使用 `logrus` 的实例对象才能实现日志分别在文件和控制台输出，避免污染全局的 `logrus`
   - 如果使用 `logrus` 将是默认的输出格式

```go
func (d *DateLogConfig) Init() *logrus.Logger {
  // 实例化 logrus
  log := logrus.New()
  // 设置是否输出文件名和行号信息
  log.SetReportCaller(true)
  // 将 logrus 的默认输出丢弃，确保日志只通过 hooks 输出
  log.SetOutput(io.Discard)

  // 控制台输出的 hook
  consoleHook := &ConsoleHook{
    formatter: &LogConsoleFormatter{},
  }
  // 添加控制台输出的 hook
  log.AddHook(consoleHook)

  // 文件路径
  filename := fmt.Sprintf("%s/%s/%s.log", d.Path, d.Date, d.Name)
  // 创建目录
  if err := os.MkdirAll(fmt.Sprintf("%s/%s", d.Path, d.Date), os.ModePerm); err != nil {
    log.Fatal(err)
  }
  // 打开文件，如果文件不存在，则创建文件
  file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
  if err != nil {
    log.Fatal(err)
  }

  // 文件输出的 hook
  fileHook := &FileHook{
    formatter: &LogFileFormatter{},
    file:      file,
  }
  // 添加文件输出的 hook
  log.AddHook(fileHook)
  return log
}
```

## 按照 level 分割

日志分为 `INFO`、` WARN`、`ERROR `、`DEBUG` 等级别，我们可以按照不同的级别将日志输出到不同的文件中

按 `level` 分割日志的实现方式和按日期分割日志类似，只是需要根据不同的日志级别创建不同的文件

### LevelFormatter

`LevelFormatter` 格式化结构体和 `LogFileFormatter` 一样

```go
type LevelFormatter struct{}

func (l *LevelFormatter) Format(entry *logrus.Entry) ([]byte, error) {
  timestamp := time.Now().Local().Format("2006-01-02 15:04:05")
  var file string
  var len int
  if entry.Caller != nil {
    file = filepath.Base(entry.Caller.File)
    len = entry.Caller.Line
  }
  msg := fmt.Sprintf("[%s] %s [%s:%d] %s\n", strings.ToUpper(entry.Level.String()), timestamp, file, len, entry.Message)
  return []byte(msg), nil
}
```

### hook

`LevelHook` 结构体和 `FileHook` 类似，只是多了几个字段，用来存储不同级别的日志文件

`Fire` 方法中，需要根据不同的日志级别将日志内容写入到不同的文件中，其他都是一样的

```go
type LevelHook struct {
  formatter logrus.Formatter
  file      *os.File
  errFile   *os.File
  warnFile  *os.File
  infoFile  *os.File
  debugFile *os.File
}

func (l *LevelHook) Levels() []logrus.Level {
  return logrus.AllLevels
}

func (l *LevelHook) Fire(entry *logrus.Entry) error {
  originalFormatter := entry.Logger.Formatter
  entry.Logger.Formatter = l.formatter
  defer func() { entry.Logger.Formatter = originalFormatter }()
  line, err := entry.Logger.Formatter.Format(entry)
  if err != nil {
    return err
  }

  // 所有的日志都写入到默认的文件中
  _, err = l.file.Write(line)

  // 根据不同的日志级别将日志内容写入到不同的文件中
  switch entry.Level {
  case logrus.ErrorLevel:
    _, err = l.errFile.Write(line)
  case logrus.WarnLevel:
    _, err = l.warnFile.Write(line)
  case logrus.InfoLevel:
    _, err = l.infoFile.Write(line)
  case logrus.DebugLevel:
    _, err = l.debugFile.Write(line)
  }
  return err
}
```

### 初始化

初始化也是一样的，唯一的区别是创建不同级别的文件，具体内容如下：

```go
type LevelConfig struct {
  Date string
  Name string
  Path string
}

func NewLevelLog(d *LevelConfig) *LevelConfig {
  return &LevelConfig{
    Date: d.Date,
    Path: d.Path,
    Name: d.Name,
  }
}

func (l *LevelConfig) Init() *logrus.Logger {
  log := logrus.New()
  log.SetReportCaller(true)
  log.SetOutput(io.Discard)

  err := os.MkdirAll(fmt.Sprintf("%s/%s", l.Path, l.Date), os.ModePerm)
  if err != nil {
    log.Fatal(err)
  }

  // 创建不同级别的文件
  allFile, _ := os.OpenFile(fmt.Sprintf("%s/%s/%s-%s.log", l.Path, l.Date, l.Name, allLog), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
  errFile, _ := os.OpenFile(fmt.Sprintf("%s/%s/%s-%s.log", l.Path, l.Date, l.Name, errLog), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
  warnFile, _ := os.OpenFile(fmt.Sprintf("%s/%s/%s-%s.log", l.Path, l.Date, l.Name, warnLog), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
  infoFile, _ := os.OpenFile(fmt.Sprintf("%s/%s/%s-%s.log", l.Path, l.Date, l.Name, infoLog), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
  debugFile, _ := os.OpenFile(fmt.Sprintf("%s/%s/%s-%s.log", l.Path, l.Date, l.Name, debugLog), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)

  fileHook := &LevelHook{
    formatter: &LevelFormatter{},
    file:      allFile,
    errFile:   errFile,
    warnFile:  warnFile,
    infoFile:  infoFile,
    debugFile: debugFile,
  }

  log.AddHook(fileHook)
  return log
}
```

## 总结

1. 自定义内容输出：实现 `Formatter` 接口的 `Format` 方法
2. 自定义输出方式：实现 `Hook` 接口的 `Levels` 和 `Fire` 方法
3. 将 `logrus` 的默认输出丢弃，确保日志只通过 `hooks` 输出



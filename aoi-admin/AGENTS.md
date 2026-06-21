# Agent Rules：项目级规则

本文件是代码代理在 Aoi Admin / `go-scaffold` 仓库中工作的项目级长期规则。后续所有开发、重构、修复、初始化、配置、CLI、WebUI、runtime、文档、测试和工程治理任务都必须遵守本文件。面向开发者的工程文档从 `docs/README.md` 开始；AI 专用补充说明位于 `docs/ai`。

## 适用范围

- 本规则适用于本项目所有代码、配置、脚本、文档、示例、测试、构建、部署和运行态资料。
- 根目录 `AGENTS.md` 是项目级 Agent 规则唯一入口；子目录规则只能补充局部约束，不得覆盖或削弱本文件。
- 任务级要求不得绕过本规则。若用户要求与本规则冲突，必须先指出冲突并确认处理方式。

## 强制规则

- 不保留旧产物、旧入口、旧字段、旧示例、旧文档、旧逻辑、旧兼容层、deprecated 设计或临时过渡方案。
- 发现废弃设计后，必须迁移到当前设计并删除旧实现，不允许新旧双轨并存。
- CLI、WebUI、runtime、配置加载和初始化流程不得各自维护重复逻辑；共享行为必须收敛到统一实现。
- 不得为了兼容旧结构污染新架构，不得把临时分支、隐藏 fallback 或过渡开关写成长期设计。
- 不得保留空配置块、无效字段、半成品示例、无说明占位或无法验证的默认值。
- 修改代码前必须先分析现状、调用链、依赖关系和影响边界。
- 不得基于猜测修改代码；所有重构、删除和迁移必须有代码证据。
- 新实现必须统一、清晰、可维护、可测试、可扩展。
- 修改后必须清理所有相关旧引用，包括代码、配置、文档、示例、测试、构建脚本和运行手册。
- 修改后必须运行与变更范围匹配的构建、测试或静态检查；无法运行时必须说明原因和风险。

## 产品定位

- Aoi Admin 的产品定位是可运行、可扩展、可托管多条业务产品线的全栈产品平台底座，不是单一后台管理系统，也不只是 Go 后端脚手架。
- 长期产品形态是“共享主平台 + 多独立产品线”：主平台统一承载账号、权限、组织租户、配置、审计日志、插件、媒体、版本、系统管理和基础运营能力；产品线独立扩展自己的公开前台、业务后台、业务模块、领域模型和用户体验。
- 公开官网、首次安装初始化流程、`/admin` 平台后台、文档、组件体系和质量工具都必须服务该定位。当前有效表达不得把项目主定位退回“后台模板”“纯脚手架”或“前端重构临时状态”。
- 未来产品线必须复用主平台已有账号、组织、权限、配置、审计、API client、i18n、Aoi React 设计系统、测试和构建流程；不得重复实现主平台已有基础设施逻辑。
- 不得在前端凭空实现后端尚未暴露的生产能力。产品叙事可以说明方向和边界，真实生产功能必须以 Go 后端 API、配置、权限、持久化和审计能力为准。

## 配置优先

- 可变业务策略、品牌标识、产品维度、平台维度、认证安全策略、会话并发策略、Cookie/CSRF/header 名称、缓存开关、缓存 TTL、运行时默认值和部署差异不得硬编码在业务代码、前端页面、store、handler 或 service 中。
- 上述可变项必须进入配置结构、默认配置、示例配置、环境变量覆盖、route contract、受控注册表或系统配置管理，并由统一配置加载和配置管理能力接入。
- 新增或修改配置项时，必须同步更新 `internal/config` 结构与校验、配置默认值、`configs/*.example.yaml`、`configs/examples/*.example.yaml`、`deploy/config.production.example.yaml`、系统配置快照、后端 i18n 标签、相关文档和测试。
- route contract 是主系统 API 的产品归属、访问级别、权限、OpenAPI 和 system API catalog 的事实来源。不得按 path、method、字符串前缀或目录二次推断产品、权限或访问策略。
- `brand.productCode` 是主平台默认产品码来源；产品线、客户端类型、平台类型、组织上下文和缓存 key 维度必须通过配置、请求上下文或 contract 传递，不得写死为固定业务值。
- IAM service 只能依赖本包定义的最小缓存、token、授权和持久化接口；Redis、本地缓存、Hybrid 缓存和缓存 key 组装必须通过应用装配和配置注入，不得在业务核心散落具体基础设施依赖。
- 稳定协议值、HTTP 方法、数据库列名、迁移里的历史回填值、枚举类型、错误码、编译期 contract 标识和包内私有常量可以保留在代码中，但不得承载可运营、可部署、可品牌化或可按产品线变化的策略。

## 执行流程

- 先阅读相关代码、配置、文档和测试，确认当前实现事实。
- 梳理入口、调用链、数据流、配置流和初始化流程。
- 明确问题边界，区分真实缺陷、历史残留、重复实现和架构漂移。
- 制定最小但完整的修改方案，确保旧设计被迁移并删除。
- 执行修改时保持单一设计路径，不引入兼容分支、临时开关或隐藏 fallback。
- 同步更新测试、示例和文档，确保它们只描述当前有效行为。
- 删除无效引用后重新搜索确认无残留。
- 运行必要验证命令，并记录结果。

## 验收要求

- 新旧实现不存在双轨并存。
- 不存在已废弃字段、入口、配置块、示例或文档残留。
- CLI、WebUI、runtime、配置加载和初始化流程复用统一逻辑。
- 调用链清晰，职责边界明确，没有为兼容旧结构引入架构污染。
- 测试、构建或静态检查通过，或明确说明未通过原因及阻塞点。
- 代码库搜索确认相关旧引用已清理。
- 文档描述当前行为，不描述已删除设计或未来愿望。

## 输出要求

- 完成涉及修改的任务后，最终输出必须包含：问题原因、修改文件、删除内容、最终设计、验证结果。
- 不得只说明“已修复”或“已完成”；必须交代变更依据、清理范围和验证结论。

## Agent Rules 编写与更新规范

- 当用户要求编写、新增、修改、补充、写入、重构 Agent Rules，或要求把某些内容整理成 Agent Rules 时，必须先理解用户真实意图、当前语境、规则适用范围和已有规则结构。
- 不得默认采用追加提示词、补充一段、末尾加一句等方式处理 Agent Rules。
- 除非用户明确要求“只追加这一段”，否则必须输出重新整合后的完整规则版本，确保内容可直接替换使用。
- 新规则必须与已有规则合并、去重、归类和压缩，避免重复、冲突、碎片化和多处表达同一约束。
- 必须区分项目级长期规则和当前任务提示词。一次性任务要求不得写入项目级长期规则；项目长期约束必须抽象为通用规则，不得保留具体任务语境。
- Agent Rules 正文必须使用明确、强约束、可执行的规范表达，不得写成建议语气。
- 不得保留模糊表达、临时说明、重复语句、过期上下文或无意义占位。
- 输出 Agent Rules 时，只输出可直接使用的规则正文，不输出写作过程、冗长解释或任务提示词。
- 当用户表达不完整但可从上下文补全时，必须补全合理语境；存在影响规则范围、存放位置或长期约束含义的关键歧义时，必须先向用户确认。

## 项目快照

- 后端运行时：Go 1.25.7；模块名：`github.com/rei0721/go-scaffold`。
- 后端技术栈：Gin HTTP 路由、Cobra/Bubble Tea CLI、GORM、goose 迁移、Casbin IAM、Redis、Zap、存储辅助、SQL 生成、Docker 示例和 GitHub Actions。
- 前端状态：`web/app` 是 React 19 一体化前端，覆盖公开官网、首次安装向导、`/admin` 共享平台后台和未来产品线入口，使用 TypeScript、Vite、React Router Framework Mode、Tailwind CSS v4、TanStack Query、Zustand、React Hook Form、Zod、Radix UI、TanStack Table、i18next/react-i18next、lucide-react、react-markdown、remark-gfm、rehype-sanitize、gray-matter、Shiki、Vitest、React Testing Library、Playwright、ESLint flat config 和 Prettier；包管理器固定为 `pnpm@10.22.0`。
- 构建与发布形态：后端通过 `go build` 生成服务二进制；React WebUI 通过 `pnpm --dir web/app build` 生成 Go 静态托管产物 `web/app/build/client`；Go 默认从根路径 `/` 托管统一 SPA，并保留 `/api`、`/api/v1`、`/health`、`/ready` 和插件协议路径不进入 SPA fallback；Dockerfile、GitHub Actions 和 `scripts/package.py` 必须使用 React 构建产物。
- i18n 状态：当前前后端语言码和资源路径未完全对齐。后端使用 `zh-CN`、`en-US` 与 `configs/locales/**` YAML 资源；React 前端使用 `zh-CN`、`en` 与 `web/app/app/i18n/locales/*.json` 资源，并通过共享 i18n/API client 映射后端 `X-Locale`；该映射必须保留到后端 canonical locale 迁移完成。
- 默认本地后端服务：在仓库根目录运行 `go run ./cmd/aoi server`，监听 `127.0.0.1:9999`，SQLite 数据位于 `data/`。
- 默认 WebUI 开发入口：在 `web/app` 使用 `pnpm dev`；如需连接本地后端，应设置 `VITE_PUBLIC_API_BASE_URL=http://127.0.0.1:9999`。

## 源码地图

- `cmd/aoi`：进程入口和命令规格，应保持轻薄。
- `internal/app`：应用装配根，负责生命周期、重载、启动和依赖注入。
- `internal/config`：配置结构、环境变量覆盖、校验、持久化和监听。
- `internal/modules`：应用业务模块。现有模块保持稳定包名 `model`、`repository`、`service` 和 `handler`。
- `internal/modules/*/service`：应用服务和领域行为。service 定义自己需要的最小接口，不导入 `pkg`、`internal/app` 或同模块 `repository` 实现。
- `internal/modules/*/repository` 与 `internal/modules/*/infrastructure`：模块基础设施实现，用于满足 service 本地 contract。
- `internal/transport/http`：HTTP 路由装配和注册。
- `internal/transport/http/contracts.go`：主系统 HTTP route contract registry，是真实路由注册、`system_apis` catalog、权限同步和 `docs/api/openapi.yaml` 生成的单一事实来源。
- `internal/middleware`：trace、auth、i18n、CORS、recovery、logging 等传输中间件。
- `internal/ports`：共享边缘端口，供 app adapter、transport、middleware、handler 和 repository infrastructure 使用。不要把它当成宽泛的 service 层依赖。
- `pkg`：可复用基础设施包，不能依赖应用模块。
- `types`：共享常量、错误和结果辅助。
- `configs`：配置示例、默认配置和 locale；记录配置行为时优先更新示例或文档。
- `configs/locales`：后端、CLI、初始化向导和 API 响应使用的 i18n YAML 资源，按 `ui`、`api`、`validation`、`system` 命名空间拆分。
- `deploy`、`Dockerfile` 与 `scripts`：生产配置、Compose 示例、部署和发布包脚本。
- `web/app`：目标 React 一体化前端，包含公开官网、首次安装向导、`/admin` 共享平台后台和未来产品线入口；长期前端约束见 `web/app/AGENTS.md` 与 `web/app/design/rules.md`。
- `docs/ai` 与 `tools/ai`：隔离的 AI 工作区和 AI 工具配置。

## 架构角色映射

当前仓库保留既有稳定包名，不把洋葱模型术语直接作为新目录要求：

- `model` 对应 domain，承载领域数据结构、领域常量和持久化模型。
- `service` 对应 application/use case，承载应用服务、领域规则、用例编排和本地接口 contract。
- `handler` 对应 adapter，负责 HTTP/RPC/CLI 等输入输出适配，不承载业务规则。
- `repository` 与 `infrastructure` 对应 infrastructure implementation，负责实现 service 定义的接口并隔离技术细节。

## 架构边界

- `pkg` 是系统级基础能力层，只封装配置、日志、数据库、缓存、消息队列、HTTP Client、链路追踪、任务调度、存储、加密、token、迁移器等可复用组件，不承载业务逻辑。
- `pkg` 不能导入 `internal/app`、`internal/modules` 或其他业务层包，避免与业务逻辑产生双向耦合。
- `internal/app` 是应用启动与装配层，负责初始化基础能力、适配 `pkg` 实现、装配业务模块、注册传输层，并统一管理启动、停止、重载和资源关闭。
- `internal/modules` 内的业务逻辑不得直接依赖 `pkg` 的具体实现，也不得自行初始化数据库、缓存、HTTP client、SMTP client、logger、config loader 等基础组件。
- 业务核心不得感知数据库、缓存、消息队列、HTTP Client、Logger、SQL、ORM、Redis Key、MQ Topic、HTTP 框架等基础设施细节。
- service 层需要数据库、事务、外部请求、通知、授权、token、TOTP、ID、host metrics 等能力时，应在本包定义最小接口，并通过构造函数注入或应用容器传递依赖。
- repository 和 module infrastructure 可以持有具体技术实现，但应通过 service-local contract 暴露能力，并隔离 ORM、HTTP、SMTP、缓存、存储等细节。
- 基础组件的创建和生命周期管理应收敛到 `internal/app`、repository 或模块 `infrastructure`，不要绕过应用生命周期在业务核心中临时创建。
- 审查架构边界漂移时，先检查 `internal` 中对 `pkg/*` 的直接导入、service 层对同模块 `repository` 的导入，以及 `http.Client`、`smtp.`、`os.Getenv`、`database.New`、`WithExecutor` 等基础设施初始化模式。
- 发现边界漂移时，优先把初始化逻辑移回 `internal/app` 或模块 infrastructure，再通过接口、构造函数注入或应用容器把能力传给业务模块。
- 更完整的分层说明见 `docs/architecture/layers.md`；已有边界测试见 `internal/import_boundary_test.go`。

## 架构审查与重构流程

当任务要求审查或修复洋葱模型、依赖边界、基础设施解耦时，按以下顺序工作：

- 先扫描当前模块分层、包依赖关系和 `pkg` 使用位置，重点检查 `internal` 业务代码中的 `pkg/*` import、service 层基础设施初始化、反向依赖和职责混杂。
- 列出疑似漂移点，至少包含文件路径、当前依赖方式、违反的架构规则和建议修复方式。
- 按最小改动原则修复：删除业务模块对 `pkg` 的直接依赖，抽取业务所需的最小接口，通过构造函数或应用容器注入依赖。
- 将数据库、缓存、消息队列、HTTP Client、Logger、Config、任务调度等具体实现的初始化、启动、关闭和注册逻辑收敛到 `internal/app` 或模块 infrastructure。
- 保持现有业务行为、接口语义、核心流程和稳定包结构不变，除非这是修复架构漂移所必需。
- 修复后重新检查依赖方向、循环依赖和边界测试；优先运行聚焦测试，再按风险运行 `go test ./...`、`go vet ./...`、`go list ./...` 和编译命令。
- 此类任务的最终回复应包含：架构审查结论、依赖漂移问题清单、每个问题的修复方式、修改过的文件列表、修复后的依赖关系说明、残留风险或需人工确认点，以及建议补充的测试或架构约束。

## 变更边界

- 不要把 AI 产物混入 `cmd`、`internal`、`pkg` 或 `types`。
- 除非任务明确要求，不要编辑 `configs/config.yaml`、`data/`、`tmp/`、本地环境文件或生成的运行时输出。
- 记录配置行为时，优先更新 `configs/*.example.yaml`、`.env.example` 或文档。
- 保持 `pkg` 可复用，并避免导入 `internal/modules` 或 `internal/app`。
- 不要把业务逻辑放进 handler；校验、事务和领域规则应放在 service 中。
- 基础设施创建和生命周期管理应放在 `internal/app`、repository 或模块 `infrastructure` 中；service 包通过构造函数接收依赖。
- `internal/migrations` 中的迁移一旦共享，应视为 append-only。

## 代码风格

- Go 代码必须使用 `gofmt` 格式化，保持小接口、显式错误返回和清晰构造函数注入；不要用全局变量、隐式初始化或包级副作用绕过应用装配。
- Go 包命名、文件组织和导出 API 应贴合现有模块风格；新增公共类型、错误码、配置字段和响应结构时，必须同步测试和文档。
- Handler 只做输入输出适配；校验、事务、领域规则和权限语义放在 service；repository 和 infrastructure 隔离 ORM、SQL、缓存、存储和外部协议细节。
- 新增或修改主系统 HTTP API 时，必须在 `internal/transport/http/contracts.go` 的同一份 route contract 中声明 method、Gin 风格 path、访问级别、权限、summary、请求/响应 DTO 和参数；真实路由注册、`system_apis` catalog、权限同步和 OpenAPI 只能从该 contract 派生，不得新增按 path/method 二次推断的权限或目录逻辑。
- 主系统 API handler 使用的请求体 DTO 必须是可被 contract 引用的稳定 Go 类型；匿名请求结构、匿名响应结构和 `map[string]any` 只能用于无法可靠建模的动态字段，普通新增 API 必须补充明确 DTO。
- `docs/api/openapi.yaml` 是生成产物，禁止手写维护；变更主系统 API route contract 后必须运行 `go run ./cmd/aoi api openapi --output docs/api/openapi.yaml`，并保留生成结果。`GET /openapi.yaml` 是公开运行时契约接口，不得纳入 `/api/v1` API catalog、权限同步、操作记录或 SPA fallback。
- `docs/api/plugin-protocol/openapi.yaml` 是远程插件协议独立契约，不使用主系统 route contract 生成；修改插件协议时按插件协议文档和 schema 单独维护。
- React 前端使用 TypeScript、React 19、React Router Framework Mode 和 Vite 约定；格式遵循 `web/app` 配置：2 空格缩进、双引号、LF 换行，最终以 Prettier 和 ESLint flat config 为准。
- React 前端后台 API 统一通过 `app/lib/api` 的 endpoint 表和 API client，不要散落新的 `/api/v1` 字符串；服务端数据缓存使用 TanStack Query，客户端偏好和认证快照使用 Zustand。
- React 前端首次安装向导必须位于 `/setup/*`，独立于公开官网和 `/admin` shell；安装步骤、字段、驱动、选项、测试能力和完成状态必须来自后端 setup schema 与 status API，不得在前端凭空扩展生产能力。
- 首次安装向导只允许新增不提交后端的 UI-only 本地校验或确认字段（例如确认密码），此类字段不得写入 API payload、URL、日志、截图、localStorage 或 sessionStorage，也不得被描述为后端 schema 能力。
- React 前端用户可见文案必须维护在 `app/i18n` locale 资源中，不要在页面、组件、store、配置、表单 schema、表格列或 SEO helper 中硬编码展示文本。
- Aoi React 组件体系必须沉淀在 `app/components/aoi`，按 tokens、primitives、patterns、templates 分层；Radix UI 只作为可访问性 primitive，Tailwind CSS v4 只作为样式工具，shadcn/ui 只能作为实现参考或初始化辅助。
- 可见 UI 变更必须保持响应式、可访问焦点、键盘操作、触控尺寸、文本对比度和 `prefers-reduced-motion` 支持；React 前端图标优先使用 `lucide-react`。
- 旧 Nuxt/Vue/Material Web 前端源码已从当前源码树移除；不得重新创建 `web/admin` 生产入口。需要查证历史行为时，只能通过 Git 历史、当前 React 迁移记录和已保留的开发文档证据链完成。

## i18n 国际化规范

- i18n 配置和资源路径必须收敛到明确入口，不得在页面、handler、service、store、composable、server route 或脚本中新增散落的语言列表、资源目录、fallback 规则或硬编码路径。
- 后端 i18n 配置统一维护在配置文件的 `i18n` 块中，长期入口包括 `configs/config.example.yaml`、`configs/examples/*.example.yaml` 和 `deploy/config.production.example.yaml`；本地运行配置只作为派生实例，不作为规则来源。
- 后端 i18n 资源统一位于 `configs/locales/{ui,api,validation,system}/{locale}.yaml`；命名空间必须继续表达用途边界：`ui` 给 WebUI/CLI/初始化向导通用界面文案，`api` 给接口响应和业务异常，`validation` 给参数校验，`system` 给内置菜单、字典、参数、版本和品牌派生标签。
- React 前端 i18n 配置统一维护在 `web/app/app/i18n`；前端运行时文案统一位于 `web/app/app/i18n/locales/{locale}.json`；本地 Markdown 博客内容位于 `web/app/content/blog/{locale}`，所有语言版本必须保持相同 slug 或明确的 locale front matter。
- 当前未对齐状态必须显式保留在规则和交付说明中：后端支持语言是 `zh-CN`、`en-US`，后端文件是 `en-US.yaml`；前端支持语言是 `zh-CN`、`en`，前端文件是 `en.json`；React API client 负责在 WebUI locale 和后端 `X-Locale` 之间转换。
- 长期目标是前后端共享同一组 canonical locale 标识、默认语言、回退语言和支持语言清单。执行该迁移时必须一次性更新后端配置、后端资源、React i18n 配置、前端资源、Markdown 内容、`X-Locale` 传递、测试、示例和文档，并删除旧语言码映射，不得保留新旧双轨。
- 在未完成对齐前，不得把前端 `en` 当作新的后端 locale，也不得把后端 `en-US` 直接写成新的前端资源目录；需要跨端传递时只能通过现有 mapping helper，并在相关变更说明中标记该差异。
- 新增或修改用户可见文案时，必须按触达面同步资源：后端/CLI/API 文案同步 `configs/locales` 下所有支持语言和相关命名空间；React WebUI 文案同步 `web/app/app/i18n/locales/zh-CN.json` 与 `web/app/app/i18n/locales/en.json`；Markdown 博客内容同步 `web/app/content/blog/zh-CN` 与 `web/app/content/blog/en`。
- 不得把大段用户可见文案、错误消息、按钮标签、表格列名、状态提示或设置项标签硬编码在 Go、React、TypeScript 或配置构造代码中；稳定技术名词如 API Token、Redis、SMTP、HTTP 方法、协议名和路径可保留英文。
- i18n 相关变更必须运行匹配验证：后端资源或配置变更运行 `go test ./pkg/i18n ./internal/config -count=1 -mod=readonly`，触及初始化或 HTTP locale 传递时扩展到相关 `internal/app`、`internal/transport/http` 测试；React WebUI 文案变更运行 `pnpm --dir web/app lint:i18n` 和 `pnpm --dir web/app typecheck`。

## 依赖限制与依赖变更

- 后端依赖只通过 `go.mod` 和 `go.sum` 管理；验证命令必须使用 `-mod=readonly`，不得让测试、构建或运行命令隐式改写依赖文件。
- 新增 Go 依赖前必须先确认标准库、现有 `pkg` 能力或已有依赖无法满足需求；不得为了局部便利引入宽泛框架、重复基础设施或业务反向依赖。
- 前端依赖只使用 pnpm 管理，不得使用 npm、yarn 或 bun 写入依赖状态；只有在有意变更依赖时才保留对应前端目录的 `pnpm-lock.yaml` 变化。
- 不要编辑或提交 `.nuxt/`、`.output/`、`node_modules/`、`web/app/build/`、`web/app/dist/`、`build/releases/`、`tmp/`、`data/` 等生成、依赖或运行态目录，除非任务明确要求交付对应产物。
- 任何新增依赖、构建工具、脚本入口或质量工具，都必须同步更新 AGENTS、README、docs 或示例中对应的命令和验证说明。

## 运行、构建、测试与检查命令

以下命令默认在仓库根目录运行；React WebUI 命令使用 `pnpm --dir web/app`。

运行命令：

```powershell
go run ./cmd/aoi server
go run ./cmd/aoi server --config=configs/config.yaml
$env:VITE_PUBLIC_API_BASE_URL="http://127.0.0.1:9999"
pnpm --dir web/app dev --host 127.0.0.1 --port 3002
```

构建命令：

```powershell
go build -mod=readonly -o ./tmp/go-scaffold-server ./cmd/aoi
go run ./cmd/aoi api openapi --output docs/api/openapi.yaml
pnpm --dir web/app build
python scripts/package.py
docker build -t aoi-admin:local .
```

测试与静态检查命令：

```powershell
go test ./internal/config -count=1 -mod=readonly
go test ./internal/transport/http -count=1 -mod=readonly
go test ./internal/... -count=1 -mod=readonly
go test ./... -count=1 -mod=readonly
go vet ./...
golangci-lint run --config tools/ai/golangci.yml ./...
govulncheck ./...
gosec ./...
osv-scanner scan source .
pnpm --dir web/app typecheck
pnpm --dir web/app lint:i18n
pnpm --dir web/app test
pnpm --dir web/app test:e2e
git diff --check
```

聚焦变更时，先运行最近包或子系统的测试；如果变更跨 package、配置、HTTP、数据库、共享类型、WebUI 静态托管或构建路径边界，再运行完整测试套件。

## 提交前检查

- 提交前必须先运行 `git status --short` 并检查 `git diff`，确认没有混入无关文件、用户改动、生成目录、运行态数据或本地环境文件。
- Go 代码变更必须对修改过的 Go 文件运行 `gofmt`，再按影响范围运行聚焦 `go test`；跨装配、配置、HTTP、数据库、共享类型或架构边界时运行 `go test ./... -count=1 -mod=readonly`、`go vet ./...` 和后端构建。
- 主系统 HTTP API 变更必须运行 `go run ./cmd/aoi api openapi --output docs/api/openapi.yaml` 和 `go test ./internal/transport/http -count=1 -mod=readonly`，确认生成契约与提交文件一致且实际 `/api/v1` 路由都有 contract registry 条目。
- React WebUI TypeScript、路由、组件、store、API、i18n 或 Markdown 内容系统变更必须运行 `pnpm --dir web/app typecheck`；用户可见文案变更还必须运行 `pnpm --dir web/app lint:i18n`；可复用逻辑或组件行为变更按范围运行 `pnpm --dir web/app test`。
- React WebUI 可见 UI、路由守卫、认证和关键后台流程变更必须运行 `pnpm --dir web/app test:e2e` 或聚焦 Playwright/Browser 检查。
- React Router、Vite、Tailwind、Markdown 预处理、构建敏感模块或静态托管路径变更必须运行 `pnpm --dir web/app build` 并确认目标静态输出目录存在 `index.html`。
- 可见 UI 或后台工作流变更必须用 Browser/视觉检查桌面与移动端，至少覆盖 `1440x900` 和 `390x844`，最终说明检查路线、视口和残余风险。
- 提交前必须运行 `git diff --check`；安全、发布、依赖或 CI 相关变更应按风险运行 `golangci-lint`、`govulncheck`、`gosec`、`osv-scanner`、Docker 或发布包命令。
- 如必要检查因本地工具缺失、环境限制或上游问题无法运行，最终输出必须说明未运行原因、影响范围和残留风险。

## GitHub 工作流

- 远端仓库预期为 `git@github.com:rin721/aoi-admin.git`。
- 处理仓库、PR、issue、label、comment 和 PR 创建工作流时，优先使用可用的 GitHub plugin/connector。
- 本地分支 PR 发现和 Actions 日志使用 GitHub CLI：`gh auth status`、`gh pr view`、`gh pr checks` 和 `gh run view`。
- 如果 `gh auth status` 显示未登录，先请用户运行 `gh auth login`，再继续 CI 日志或 PR thread 工作流。

## AI 工作区

- `docs/ai/README.md`：当前 AI 操作入口。
- `docs/ai/project-map.md`：面向代理的紧凑架构地图。
- `docs/ai/tooling.md`：已安装工具和设置命令。
- `docs/ai/prompts.md`：常见仓库任务的可复用提示词。
- `docs/ai/handoff-template.md`：长任务简短交接模板。
- `docs/ai` 下的长任务记录是历史证据，除非 README 标明其为当前任务输入。
- `tools/ai/golangci.yml`：AI 辅助 lint 配置。
- `tools/ai/security-checks.md`：本地安全扫描运行手册。
- 短期报告放在 `tmp/ai`；`tmp/` 已被 git 忽略。

## 文档约定

- 现有文档应描述当前行为，而不是未来愿望。未来能力或缺失能力写入 `docs/backlog/known-gaps.md`；仅面向代理运行的说明写入 `docs/ai`。
- 文档注释使用中文。
- 保留具体命令、文件路径和已验证事实。
- 如果终端输出出现 mojibake，先用编辑器或原始字节检查文件，再重写大段文档。

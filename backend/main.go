package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"database/sql" 

	_ "github.com/go-sql-driver/mysql"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv" 

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	bucket    string
	region    string
	endpoint  string
	accessKey string
	secretKey string
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("❌ Ошибка загрузки .env:", err)
	} else {
		log.Println("✅ Файл .env загружен успешно")
	}

	bucket = os.Getenv("BACKBLAZE_BUCKET")
	region = os.Getenv("BACKBLAZE_REGION")
	endpoint = os.Getenv("BACKBLAZE_ENDPOINT")
	accessKey = os.Getenv("BACKBLAZE_ACCESS_KEY")
	secretKey = os.Getenv("BACKBLAZE_SECRET_KEY")

	log.Println("🔍 Проверяем загруженные переменные из .env:")
	log.Println("BACKBLAZE_BUCKET:", bucket)
	log.Println("BACKBLAZE_REGION:", region)
	log.Println("BACKBLAZE_ENDPOINT:", endpoint)
	log.Println("BACKBLAZE_ACCESS_KEY:", accessKey)
	log.Println("BACKBLAZE_SECRET_KEY:", secretKey)
	if bucket == "" || accessKey == "" || secretKey == "" {
		log.Fatalf("Ошибка: не все переменные окружения Backblaze загружены!")
	}
}

func connectDB() (*sql.DB, error) {
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	dbname := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s", user, pass, host, dbname)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}

	log.Println("Подключение к базе данных установлено")
	return db, nil
}

func hashPassword(password string) (string, error) {
	log.Println("🔒 Начало хеширования пароля...")
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Ошибка при хешировании пароля:", err)
		return "", err
	}
	log.Println("✅ Пароль успешно захеширован!")
	return string(bytes), nil
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("📩 Новый запрос на /register")

	if r.Method != http.MethodPost {
		log.Println("Ошибка: Неверный метод", r.Method)
		http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		return
	}

	log.Println("📌 Заголовки запроса:", r.Header)

	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "multipart/form-data") {
		log.Println("Ошибка: Content-Type должен быть multipart/form-data, а пришёл:", contentType)
		http.Error(w, "Неверный Content-Type", http.StatusBadRequest)
		return
	}

	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		log.Println("Ошибка в ParseMultipartForm:", err)
		http.Error(w, "Ошибка обработки формы: "+err.Error(), http.StatusBadRequest)
		return
	}

	// ⛔ Проверка галочки
	agree := r.FormValue("agreeRules")
	if agree != "on" {
		http.Error(w, "Вы должны принять правила сайта", http.StatusBadRequest)
		return
	}

	log.Println("Форма успешно разобрана!")

	var (
		username, email, password, profileName, phone, country, city, district, nationality, bodyType, about string
		eyeColor, hairColor, hairLength, breastSize, breastType, orientation, smoker, tattoo, piercing       string
		age, height, weight, price30Min, price1h, price2h, price24h                                          int

		incall, outcall, currency                       string
		priceIncall1h, priceIncall2h, priceIncall24h    int
		priceOutcall1h, priceOutcall2h, priceOutcall24h int
	)

	username = r.FormValue("name")
	email = r.FormValue("email")
	password = r.FormValue("password")
	profileName = r.FormValue("profile_name")
	phone = r.FormValue("phone")
	country = r.FormValue("country")
	city = r.FormValue("city")
	district = r.FormValue("district")
	nationality = r.FormValue("nationality")
	bodyType = r.FormValue("body_type")
	about = r.FormValue("about")

	eyeColor = r.FormValue("eye_color")
	hairColor = r.FormValue("hair_color")
	hairLength = r.FormValue("hair_length")

	hairLength = r.FormValue("hair_length")

	breastSize = r.FormValue("breast_size")
	breastType = r.FormValue("breast_type")

	breastType = r.FormValue("breast_type")

	intim := r.FormValue("intim")
	log.Println("🪒 Интимная стрижка:", intim)

	breastTypeMapping := map[string]string{
		"Натуральная": "Natural",
		"Силиконовая": "Silicone",
	}

	if engValue, exists := breastTypeMapping[breastType]; exists {
		breastType = engValue
	} else {
		log.Println("❌ Ошибка: Недопустимое значение для breast_type:", breastType)
		http.Error(w, "Ошибка: Некорректный тип груди", http.StatusBadRequest)
		return
	}

	orientation = r.FormValue("orientation")

	orientationMapping := map[string]string{
		"Гетеро":            "Heterosexual",
		"Гетеросексуальная": "Heterosexual",
		"Би":                "Bisexual",
		"Бисексуальная":     "Bisexual",
		"Лесби":             "Lesbian",
		"Лесбийская":        "Lesbian",
	}

	if engValue, exists := orientationMapping[orientation]; exists {
		orientation = engValue
	} else {
		log.Println("❌ Ошибка: Недопустимое значение для orientation:", orientation)
		http.Error(w, "Ошибка: Некоррентная ориентация", http.StatusBadRequest)
		return
	}

	smoker = r.FormValue("smoker")
	tattoo = r.FormValue("tattoo")
	piercing = r.FormValue("piercing")

	incall = r.FormValue("incall")
	if incall == "true" {
		incall = "1"
	} else {
		incall = "0"
	}

	outcall = r.FormValue("outcall")
	if outcall == "true" {
		outcall = "1"
	} else {
		outcall = "0"
	}

	currency = r.FormValue("currency")

	priceIncall1h, _ = strconv.Atoi(strings.TrimLeft(r.FormValue("price_incall_1h"), "0"))
	priceIncall2h, _ = strconv.Atoi(strings.TrimLeft(r.FormValue("price_incall_2h"), "0"))
	priceIncall24h, _ = strconv.Atoi(strings.TrimLeft(r.FormValue("price_incall_24h"), "0"))

	priceOutcall1h, _ = strconv.Atoi(strings.TrimLeft(r.FormValue("price_outcall_1h"), "0"))
	priceOutcall2h, _ = strconv.Atoi(strings.TrimLeft(r.FormValue("price_outcall_2h"), "0"))
	priceOutcall24h, _ = strconv.Atoi(strings.TrimLeft(r.FormValue("price_outcall_24h"), "0"))

	rusToDbBool := map[string]string{
		"Да":  "1",
		"Нет": "0",
	}

	if val, ok := rusToDbBool[r.FormValue("smoker")]; ok {
		smoker = val
	} else {
		smoker = "0"
	}

	if val, ok := rusToDbBool[r.FormValue("tattoo")]; ok {
		tattoo = val
	} else {
		tattoo = "0"
	}

	if val, ok := rusToDbBool[r.FormValue("piercing")]; ok {
		piercing = val
	} else {
		piercing = "0"
	}

	languagesMap := map[string]string{
		"georgian":    r.FormValue("languages[georgian]"),
		"russian":     r.FormValue("languages[russian]"),
		"english":     r.FormValue("languages[english]"),
		"azerbaijani": r.FormValue("languages[azerbaijani]"),
	}

	messenger := strings.Join(r.Form["messenger[]"], ",")
	log.Println("📌 Messenger:", messenger)
	features := strings.Join(r.Form["features[]"], ",")
	meetingFormat := strings.Join(r.Form["meeting_format[]"], ",")

	log.Println("📌 Messenger:", messenger)
	log.Println("📌 Features:", features)
	log.Println("📌 Meeting Format:", meetingFormat)

	var conversionErr error
	age, conversionErr = strconv.Atoi(r.FormValue("age"))

	heightStr := r.FormValue("height")
	if heightStr != "" {
		if h, err := strconv.Atoi(heightStr); err == nil {
			height = h
		} else {
			log.Println("Ошибка преобразования роста:", err)
			height = 0
		}
	}

	weightStr := r.FormValue("weight")
	if weightStr != "" {
		if w, err := strconv.Atoi(weightStr); err == nil {
			weight = w
		} else {
			log.Println("Ошибка преобразования веса:", err)
			weight = 0
		}
	}

	if conversionErr != nil {
		log.Println("Ошибка конвертации возраста:", conversionErr)
		http.Error(w, "Некорректный возраст", http.StatusBadRequest)
		return
	}

	price30Min, _ = strconv.Atoi(strings.TrimLeft(r.FormValue("price_30min"), "0"))
	price1h, _ = strconv.Atoi(strings.TrimLeft(r.FormValue("price_1h"), "0"))
	price2h, _ = strconv.Atoi(strings.TrimLeft(r.FormValue("price_2h"), "0"))
	price24h, _ = strconv.Atoi(strings.TrimLeft(r.FormValue("price_24h"), "0"))

	hashedPassword, err := hashPassword(password)
	if err != nil {
		log.Println("Ошибка хеширования пароля:", err)
		http.Error(w, "Ошибка обработки данных", http.StatusInternalServerError)
		return
	}

	db, err := connectDB()
	if err != nil {
		log.Println("Ошибка подключения к БД:", err)
		http.Error(w, "Ошибка подключения к базе данных", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var existingEmail string
	err = db.QueryRow("SELECT email FROM profiles WHERE email = ?", email).Scan(&existingEmail)

	if err == nil {
		log.Println("Ошибка: email уже существует в базе:", email)
		http.Error(w, "Этот email уже зарегистрирован", http.StatusConflict)
		return
	} else if err != sql.ErrNoRows { 
		log.Println("Ошибка при проверке email:", err)
		http.Error(w, "Ошибка проверки email", http.StatusInternalServerError)
		return
	}

	var uploadedPhotoURLs []string
	var uploadedVideoURLs []string

	photos, ok := r.MultipartForm.File["photos[]"]
	if !ok || len(photos) == 0 {
		log.Println("Ошибка: Фото не загружены")
	} else {
		for _, photoHeader := range photos {
			photoFile, err := photoHeader.Open()
			if err != nil {
				log.Println("Ошибка открытия фото:", err)
				continue
			}
			defer photoFile.Close()

			uniqueFileName := uuid.New().String() + filepath.Ext(photoHeader.Filename)

			fileURL, err := uploadFileToBackblaze(photoFile, uniqueFileName)
			if err != nil {
				log.Println("Ошибка загрузки фото в Backblaze:", err)
				continue
			}

			uploadedPhotoURLs = append(uploadedPhotoURLs, fileURL)
			log.Println("Фото загружено:", fileURL)
		}
	}

	videos := r.MultipartForm.File["videos[]"]
	for _, videoHeader := range videos {
		videoFile, err := videoHeader.Open()
		if err != nil {
			log.Println("Ошибка открытия видео:", err)
			continue
		}
		defer videoFile.Close()

		uniqueFileName := uuid.New().String() + filepath.Ext(videoHeader.Filename)

		fileURL, err := uploadFileToBackblaze(videoFile, uniqueFileName)
		if err != nil {
			log.Println("Ошибка загрузки видео в Backblaze:", err)
			continue
		}

		uploadedVideoURLs = append(uploadedVideoURLs, fileURL)
	}

	query := `
INSERT INTO profiles (
    username, email, password_hash, profile_name, phone, age, country, 
    city, district, nationality, body_type, eye_color,
    hair_color, hair_length, breast_size, breast_type, orientation,
    smoker, tattoo, piercing, intim, languages, about,
    price_30min, price_1h, price_2h, price_24h,
    incall, outcall, currency,
    price_incall_1h, price_incall_2h, price_incall_24h,
    price_outcall_1h, price_outcall_2h, price_outcall_24h,
    height, weight,
    messenger, features, meeting_format, status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'Hold')
`

	languagesJSON, err := json.Marshal(languagesMap)
	if err != nil {
		log.Println("Ошибка при формировании languages JSON:", err)
		http.Error(w, "Ошибка обработки языков", http.StatusInternalServerError)
		return
	}

	res, err := db.Exec(query,
		username, email, hashedPassword, profileName, phone, age, country,
		city, district, nationality, bodyType, eyeColor,
		hairColor, hairLength, breastSize, breastType, orientation,
		smoker, tattoo, piercing, intim, string(languagesJSON), about,
		price30Min, price1h, price2h, price24h,
		incall, outcall, currency,
		priceIncall1h, priceIncall2h, priceIncall24h,
		priceOutcall1h, priceOutcall2h, priceOutcall24h,
		height, weight,
		messenger, features, meetingFormat)

	if err != nil {
		log.Println("Ошибка SQL-запроса:", err)
		http.Error(w, "Ошибка сохранения анкеты", http.StatusInternalServerError)
		return
	}

	lastInsertID, err := res.LastInsertId()
	if err != nil {
		log.Println("Ошибка получения ID новой анкеты:", err)
		http.Error(w, "Ошибка регистрации", http.StatusInternalServerError)
		return
	}

	for _, service := range r.Form["services[]"] {
		_, err = db.Exec("INSERT INTO services (profile_id, service_name, included) VALUES (?, ?, 1)", lastInsertID, service)
		if err != nil {
			log.Println("Ошибка при добавлении услуги:", service, err)
		} else {
			log.Println("Услуга успешно добавлена:", service)
		}
	}

	if len(uploadedPhotoURLs) > 0 {
		for _, photoURL := range uploadedPhotoURLs {
			_, err = db.Exec("INSERT INTO profile_photos (profile_id, photo_url) VALUES (?, ?)", lastInsertID, photoURL)
			if err != nil {
				log.Println("Ошибка при добавлении фото в БД:", err)
			}
		}
	} else {
		log.Println("Фото не загружены")
	}

	if len(uploadedVideoURLs) > 0 {
		for _, videoURL := range uploadedVideoURLs {
			_, err = db.Exec("INSERT INTO profile_videos (profile_id, video_url) VALUES (?, ?)", lastInsertID, videoURL)
			if err != nil {
				log.Println("Ошибка при добавлении видео в БД:", err)
			}
		}
	} else {
		log.Println("Видео не загружены")
	}

	log.Println("✅ Анкета успешно создана:", username)
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"status": "success", "message": "Анкета успешно создана"}`)

}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	email := r.FormValue("email")

	if email == "" {
		log.Println("❌ Ошибка: Email не передан в запросе")
		http.Error(w, "Email обязателен", http.StatusBadRequest)
		return
	}

	password := r.FormValue("password")
	log.Println("🚀 Пароль, который мы получаем с фронтенда:", password)
	log.Println("Введённый пароль:", password)

	password = strings.TrimSpace(password)

	db, err := connectDB()
	if err != nil {
		log.Println("Ошибка подключения к БД:", err)
		http.Error(w, "Ошибка подключения к базе данных", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var storedHash string
	err = db.QueryRow("SELECT password_hash FROM profiles WHERE email = ?", email).Scan(&storedHash)
	if err == sql.ErrNoRows {
		log.Println("Ошибка: Email не зарегистрирован:", email)
		http.Error(w, "Email не найден", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println("Ошибка при проверке пользователя в БД:", err)
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	log.Println("Хэш из базы:", storedHash)

	log.Printf("➡️ Длина хэша: %d\n", len(storedHash))
	log.Printf("➡️ Длина пароля: %d\n", len(password))
	log.Printf("➡️ Байты хэша: %q\n", storedHash)
	log.Printf("➡️ Байты пароля: %q\n", password)

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err != nil {
		log.Println("Ошибка: Введён неверный пароль")
		http.Error(w, "Неверный email или пароль", http.StatusUnauthorized)
		return
	}

	log.Println("Вход успешен!")

	clientIP := r.RemoteAddr
	log.Println("Успешный вход:", email, "с IP:", clientIP)

	http.SetCookie(w, &http.Cookie{
		Name:   "user_email",
		Value:  email,
		Path:   "/",
		MaxAge: 3600 * 24 * 7, 
	})

	http.Redirect(w, r, "/account.html", http.StatusSeeOther)

}

func profilesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Получен запрос на /profiles")

	db, err := connectDB()
	if err != nil {
		log.Println("Ошибка подключения к БД:", err)
		http.Error(w, "Ошибка подключения к базе данных", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	city := r.URL.Query().Get("city")
	log.Println("Запрошен город:", city)

	query := `
SELECT p.id, p.username, p.age, p.height, p.weight, p.hair_color, p.nationality,
       p.price_1h, p.price_2h, p.price_24h, p.country, p.city, p.district, p.last_active, p.verified, p.messenger, p.main_photo_url
FROM profiles p
JOIN profile_payments pay ON pay.profile_id = p.id
WHERE p.status = 'Active' AND p.status != 'Lux Queen' AND p.frozen = 0
  AND pay.active_until >= CURDATE()
  AND pay.frozen = 0
`

	var args []interface{}

	if city != "" {
		query += " AND p.city = ?"
		args = append(args, city)
	}

	query += `
ORDER BY
    p.up_timestamp DESC,
    p.last_active DESC,
    p.id DESC
`

	rows, err := db.Query(query, args...)

	if err != nil {
		log.Println("Ошибка выполнения SQL-запроса:", err)
		http.Error(w, "Ошибка получения анкет", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var profiles []map[string]interface{}

	for rows.Next() {
		var id, age, price1h, price2h, price24h int
		var height, weight sql.NullInt64
		var username, hairColor, nationality, country, city, district string
		var lastActiveStr string
		var verifiedInt int
		var messenger sql.NullString
		var mainPhoto sql.NullString

		if err := rows.Scan(&id, &username, &age, &height, &weight, &hairColor, &nationality,
			&price1h, &price2h, &price24h, &country, &city, &district, &lastActiveStr, &verifiedInt, &messenger, &mainPhoto); err != nil {

			log.Println("Ошибка чтения данных из БД:", err)
			continue
		}

		layout := "2006-01-02 15:04:05"
		parsedTime, err := time.Parse(layout, lastActiveStr)
		if err != nil {
			log.Println("Ошибка парсинга даты last_active:", err)
			parsedTime = time.Time{}
		}
		isOnline := time.Since(parsedTime) <= 5*time.Minute

		photoRows, err := db.Query(`SELECT photo_url FROM profile_photos WHERE profile_id = ? ORDER BY id DESC`, id)
		if err != nil {
			log.Println("⚠️ Ошибка загрузки фото для профиля", id, ":", err)
			continue
		}

		var photos []string
		for photoRows.Next() {
			var url string
			if err := photoRows.Scan(&url); err == nil {
				photos = append(photos, url)
			}
		}
		defer photoRows.Close()

		if len(photos) == 0 {
			photos = append(photos, "/static/images/default.jpg")
		}

		var messengers []string
		if messenger.Valid && messenger.String != "" {
			parts := strings.Split(messenger.String, ",")
			for _, m := range parts {
				m = strings.TrimSpace(m)
				if m != "" {
					messengers = append(messengers, m)
				}
			}
		}

		profiles = append(profiles, map[string]interface{}{
			"id":             id,
			"username":       username,
			"age":            age,
			"height":         height.Int64,
			"weight":         weight.Int64,
			"hair_color":     hairColor,
			"nationality":    nationality,
			"price_1h":       price1h,
			"price_2h":       price2h,
			"price_24h":      price24h,
			"country":        country,
			"city":           city,
			"district":       district,
			"photos":         photos,
			"online":         isOnline,
			"verified":       verifiedInt == 1,
			"last_active":    lastActiveStr,
			"messengers":     messengers, 
			"main_photo_url": mainPhoto.String,
		})
	}

	if err = rows.Err(); err != nil {
		log.Println("Ошибка при переборе строк БД:", err)
		http.Error(w, "Ошибка обработки данных", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profiles)
}

type Service struct {
	Name       string
	Included   bool
	ExtraPrice sql.NullInt64
}

type Profile struct {
	ID              int
	Name            string
	ProfileName     string
	Age             int
	Country         string
	City            string
	District        string
	About           string
	Phone           string
	Nationality     string
	Currency        string
	Height          sql.NullInt64
	Weight          sql.NullInt64
	BodyType        sql.NullString
	Languages       sql.NullString
	LanguagesList   []string
	Incall          sql.NullBool
	Outcall         sql.NullBool
	Price30m        sql.NullInt64
	Price1h         sql.NullInt64
	Price2h         sql.NullInt64
	Price12h        sql.NullInt64
	Price24h        sql.NullInt64
	PriceIncall1h   sql.NullInt64
	PriceIncall2h   sql.NullInt64
	PriceIncall24h  sql.NullInt64
	PriceOutcall1h  sql.NullInt64
	PriceOutcall2h  sql.NullInt64
	PriceOutcall24h sql.NullInt64
	Messenger       sql.NullString
	Messengers      []string 
	Features        sql.NullString
	MeetingFormat   sql.NullString
	EyeColor        sql.NullString
	HairColor       sql.NullString
	HairLength      sql.NullString
	BustSize        sql.NullString
	BustType        sql.NullString
	Orientation     sql.NullString
	Intim           sql.NullString `json:"intim"`
	Smoker          sql.NullBool
	Tattoo          sql.NullBool
	Piercing        sql.NullBool
	Photos          []string
	Videos          []string
	Services        []Service
	Online          bool
	Verified        bool
	ViewsTotal      int
	ViewsToday      int
	ViewsTodayDate  string
}

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	profileID := strings.TrimPrefix(r.URL.Path, "/profile/")
	if profileID == "" {
		log.Println("Ошибка: profileID отсутствует в URL-пути")
		http.Error(w, "Profile ID is required", http.StatusBadRequest)
		return
	}

	profileIDInt, err := strconv.Atoi(profileID)
	if err != nil {
		log.Println("Ошибка конвертации profileID в число:", err)
		http.Error(w, "Invalid profile ID", http.StatusBadRequest)
		return
	}

	db, err := connectDB()
	if err != nil {
		log.Println("Ошибка подключения к БД:", err)
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var profile Profile
	var lastActive sql.NullTime 

	err = db.QueryRow(`
    SELECT id, username, profile_name, phone, country, city, district, age, nationality, height, weight,
           body_type, eye_color, hair_color, hair_length, breast_size, breast_type, orientation,
           smoker, tattoo, piercing, currency, languages, about, intim, price_1h, price_2h, price_24h,
           messenger, features, meeting_format,
           price_incall_1h, price_incall_2h, price_incall_24h,
           price_outcall_1h, price_outcall_2h, price_outcall_24h,
	   views_total, views_today, views_today_date,
	   verified, online, last_active

    FROM profiles WHERE id = ?`, profileIDInt).Scan(
		&profile.ID, &profile.Name, &profile.ProfileName, &profile.Phone, &profile.Country, &profile.City, &profile.District,
		&profile.Age, &profile.Nationality, &profile.Height, &profile.Weight, &profile.BodyType,
		&profile.EyeColor, &profile.HairColor, &profile.HairLength, &profile.BustSize, &profile.BustType,
		&profile.Orientation, &profile.Smoker, &profile.Tattoo, &profile.Piercing, &profile.Currency, &profile.Languages,
		&profile.About, &profile.Intim, &profile.Price1h, &profile.Price2h, &profile.Price24h,
		&profile.Messenger, &profile.Features, &profile.MeetingFormat,
		&profile.PriceIncall1h, &profile.PriceIncall2h, &profile.PriceIncall24h,
		&profile.PriceOutcall1h, &profile.PriceOutcall2h, &profile.PriceOutcall24h,
		&profile.ViewsTotal, &profile.ViewsToday, &profile.ViewsTodayDate,
		&profile.Verified, &profile.Online, &lastActive,
	)

	log.Println("Last Active (raw):", lastActive)

	profile.Currency = FormatCurrencySymbol(profile.Currency)

	if profile.Languages.Valid {
		var langs map[string]string
		err = json.Unmarshal([]byte(profile.Languages.String), &langs)
		if err != nil {
			log.Println("Ошибка разбора JSON языков:", err)
		} else {
			for lang, level := range langs {
				if level != "" {
					profile.LanguagesList = append(profile.LanguagesList, fmt.Sprintf("%s: %s", lang, level))
				}
			}
		}
	}

	if profile.Messenger.Valid && profile.Messenger.String != "" {
		parts := strings.Split(profile.Messenger.String, ",")
		for i := range parts {
			parts[i] = strings.ToLower(strings.TrimSpace(parts[i]))
		}
		profile.Messengers = parts
	}

	if err == sql.ErrNoRows {
		log.Println("Профиль не найден:", profileIDInt)
		http.Error(w, "Профиль не найден", http.StatusNotFound)
		return
	} else if err != nil {
		log.Println("Ошибка в запросе к БД:", err)
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	photoRows, err := db.Query(`SELECT photo_url FROM profile_photos WHERE profile_id = ? ORDER BY id DESC`, profileIDInt)
	if err != nil {
		log.Println("Ошибка загрузки фотографий:", err)
		http.Error(w, "Ошибка загрузки фотографий", http.StatusInternalServerError)
		return
	}
	defer photoRows.Close()

	for photoRows.Next() {
		var photoURL string
		if err := photoRows.Scan(&photoURL); err != nil {
			log.Println("Ошибка чтения фото из БД:", err)
			continue
		}
		profile.Photos = append(profile.Photos, photoURL)
	}
	if len(profile.Photos) == 0 {
		profile.Photos = append(profile.Photos, "default.jpg")
	}

	videoRows, err := db.Query(`SELECT video_url FROM profile_videos WHERE profile_id = ? ORDER BY id DESC`, profileIDInt)
	if err != nil {
		log.Println("Ошибка загрузки видео:", err)
		http.Error(w, "Ошибка загрузки видео", http.StatusInternalServerError)
		return
	}
	defer videoRows.Close()

	for videoRows.Next() {
		var videoURL string
		if err := videoRows.Scan(&videoURL); err != nil {
			log.Println("Ошибка чтения видео из БД:", err)
			continue
		}
		profile.Videos = append(profile.Videos, videoURL)
	}

	serviceRows, err := db.Query(`SELECT service_name, included, extra_price FROM services WHERE profile_id = ?`, profileIDInt)
	if err != nil {
		log.Println("❌ Ошибка загрузки услуг:", err)
		http.Error(w, "Ошибка загрузки услуг", http.StatusInternalServerError)
		return
	}
	defer serviceRows.Close()

	for serviceRows.Next() {
		var service Service
		if err = serviceRows.Scan(&service.Name, &service.Included, &service.ExtraPrice); err != nil {
			log.Println("Ошибка чтения услуги из БД:", err)
			continue
		}
		profile.Services = append(profile.Services, service)
	}

	log.Printf("Данные для шаблона: %+v\n", profile)

	log.Printf("Online: %v | Verified: %v", profile.Online, profile.Verified)

	tmpl := template.New("profile.html").Funcs(template.FuncMap{
		"lower": strings.ToLower,
	})
	tmpl, err = tmpl.ParseFiles("/var/www/luxegirlsclub.com/templates/profile.html")
	if err != nil {
		log.Println("Ошибка загрузки шаблона profile.html:", err)
		http.Error(w, "Ошибка сервера: не удалось загрузить шаблон", http.StatusInternalServerError)
		return
	}

	if profile.Photos == nil {
		profile.Photos = []string{"default.jpg"} 
	}
	if profile.Videos == nil {
		profile.Videos = []string{} 
	}

	cleanPhone := strings.NewReplacer("+", "", " ", "", "-", "", "(", "", ")", "").Replace(profile.Phone)

	if lastActive.Valid && time.Since(lastActive.Time) < 5*time.Minute {
		profile.Online = true
	} else {
		profile.Online = false
	}

	data := struct {
		Profile    Profile
		Photos     []string
		Videos     []string
		Services   []Service
		CleanPhone string
	}{
		Profile:    profile,
		Photos:     profile.Photos,
		Videos:     profile.Videos,
		Services:   profile.Services,
		CleanPhone: cleanPhone,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Println("Ошибка рендеринга страницы:", err)
		http.Error(w, "Ошибка рендера", http.StatusInternalServerError)
		return
	}

	log.Println("Профиль успешно отрендерен:", profileID)

}

func getBackblazeConfig() (string, string, string, string, error) {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Ошибка загрузки .env:", err)
		return "", "", "", "", fmt.Errorf("ошибка загрузки .env")
	}

	bucket := os.Getenv("BACKBLAZE_BUCKET")
	endpoint := os.Getenv("BACKBLAZE_ENDPOINT")
	accessKey := os.Getenv("BACKBLAZE_ACCESS_KEY")
	secretKey := os.Getenv("BACKBLAZE_SECRET_KEY")

	if bucket == "" || endpoint == "" || accessKey == "" || secretKey == "" {
		log.Println("Ошибка: не все переменные окружения Backblaze заданы")
		return "", "", "", "", fmt.Errorf("не все переменные окружения Backblaze заданы")
	}

	return bucket, endpoint, accessKey, secretKey, nil
}

func uploadFileToBackblaze(file io.Reader, fileName string) (string, error) {
	var err error
	var size int64

	// Читаем файл в буфер
	fileBuffer := bytes.NewBuffer(nil)
	size, err = io.Copy(fileBuffer, file)
	if err != nil {
		log.Println("Ошибка чтения файла в буфер:", err)
		return "", err
	}

	if fileBuffer.Len() == 0 {
		log.Println("Файл пустой — загрузка невозможна")
		return "", fmt.Errorf("файл пустой, загрузка невозможна")
	}

	log.Printf("Размер файла: %d байт", size)

	reader := bytes.NewReader(fileBuffer.Bytes())

	bucket := os.Getenv("BACKBLAZE_BUCKET")
	endpoint := os.Getenv("BACKBLAZE_ENDPOINT")
	accessKey := os.Getenv("BACKBLAZE_ACCESS_KEY")
	secretKey := os.Getenv("BACKBLAZE_SECRET_KEY")

	var sess *session.Session
	sess, err = session.NewSession(&aws.Config{
		Region:           aws.String("us-west-002"),
		Endpoint:         aws.String(endpoint),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		log.Println("Ошибка создания AWS-сессии:", err)
		return "", err
	}

	svc := s3.New(sess)

	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(fileName),
		Body:   reader,
	}

	log.Println("📤 Загружаем файл в Backblaze через S3 API...")

	_, err = svc.PutObject(input)
	if err != nil {
		log.Printf("Ошибка загрузки файла в Backblaze: %v\n", err)
		return "", err
	}

	fileURL := fmt.Sprintf("%s/%s/%s", endpoint, bucket, fileName)
	log.Printf("Файл успешно загружен: %s\n", fileURL)

	return fileURL, nil

}

func myProfileHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Запрос данных профиля пользователя")

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		http.Error(w, `{"status":"error","message":"Email обязателен"}`, http.StatusBadRequest)
		return
	}
	email := req.Email

	db, err := connectDB()
	if err != nil {
		log.Println("Ошибка подключения к БД:", err)
		http.Error(w, `{"error": "Ошибка базы данных"}`, http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var profile Profile

	err = db.QueryRow(`
    SELECT id, username, profile_name, phone, city, district, age, nationality, height, weight,
           body_type, eye_color, hair_color, hair_length, breast_size, breast_type, orientation,
           smoker, tattoo, piercing, languages, services, about, price_1h, price_2h, price_24h, messenger, features, meeting_format
    FROM profiles WHERE email = ?`, email).Scan(
		&profile.ID, &profile.Name, &profile.ProfileName, &profile.Phone, &profile.City, &profile.District,
		&profile.Age, &profile.Nationality, &profile.Height, &profile.Weight, &profile.BodyType,
		&profile.EyeColor, &profile.HairColor, &profile.HairLength, &profile.BustSize, &profile.BustType,
		&profile.Orientation, &profile.Smoker, &profile.Tattoo, &profile.Piercing, &profile.Languages,
		&profile.Services, &profile.About, &profile.Price1h, &profile.Price2h, &profile.Price24h,
		&profile.Messenger, &profile.Features, &profile.MeetingFormat,
	)

	if err != nil {
		log.Println("Ошибка загрузки данных профиля:", err)
		http.Error(w, `{"error": "Профиль не найден"}`, http.StatusNotFound)
		return
	}

	log.Println("Найден профиль ID:", profile.ID)

	photoRows, err := db.Query("SELECT photo_url FROM profile_photos WHERE profile_id = ?", profile.ID)
	if err != nil {
		log.Println("Ошибка загрузки фото:", err)
	} else {
		defer photoRows.Close()
		for photoRows.Next() {
			var photo string
			if err := photoRows.Scan(&photo); err == nil {
				profile.Photos = append(profile.Photos, photo)
			}
		}
	}

	videoRows, err := db.Query("SELECT video_url FROM profile_videos WHERE profile_id = ?", profile.ID)
	if err != nil {
		log.Println("Ошибка загрузки видео:", err)
	} else {
		defer videoRows.Close()
		for videoRows.Next() {
			var video string
			if err := videoRows.Scan(&video); err == nil {
				profile.Videos = append(profile.Videos, video)
			}
		}
	}

	response := map[string]interface{}{
		"id":             profile.ID,
		"username":       profile.Name,
		"profile_name":   profile.ProfileName,
		"phone":          profile.Phone,
		"city":           profile.City,
		"district":       profile.District,
		"nationality":    profile.Nationality,
		"body_type":      profile.BodyType,
		"eye_color":      profile.EyeColor,
		"hair_color":     profile.HairColor,
		"hair_length":    profile.HairLength,
		"breast_size":    profile.BustSize,
		"bust_type":      profile.BustType,
		"orientation":    profile.Orientation,
		"smoker":         profile.Smoker,
		"tattoo":         profile.Tattoo,
		"piercing":       profile.Piercing,
		"age":            profile.Age,
		"height":         profile.Height,
		"weight":         profile.Weight,
		"languages":      profile.Languages,
		"services":       profile.Services,
		"about":          profile.About,
		"price_1h":       profile.Price1h,
		"price_2h":       profile.Price2h,
		"price_24h":      profile.Price24h,
		"messenger":      profile.Messenger,
		"features":       profile.Features,
		"meeting_format": profile.MeetingFormat,
		"photos":         profile.Photos,
		"videos":         profile.Videos,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func deleteFileFromBackblaze(fileName string) error {
	sess, err := session.NewSession(&aws.Config{
		Region:   aws.String(os.Getenv("BACKBLAZE_REGION")),
		Endpoint: aws.String(os.Getenv("BACKBLAZE_ENDPOINT")),
		Credentials: credentials.NewStaticCredentials(
			os.Getenv("BACKBLAZE_ACCESS_KEY"),
			os.Getenv("BACKBLAZE_SECRET_KEY"),
			"",
		),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		log.Println("Ошибка создания AWS-сессии:", err)
		return err
	}

	svc := s3.New(sess)

	_, err = svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(os.Getenv("BACKBLAZE_BUCKET")),
		Key:    aws.String(fileName),
	})
	if err != nil {
		log.Println("Ошибка удаления файла из Backblaze B2:", err)
		return err
	}

	log.Println("Файл успешно удалён из Backblaze B2:", fileName)
	return nil
}

func deletePhotoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	type RequestData struct {
		PhotoURL string `json:"photo"`
	}

	var req RequestData
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.PhotoURL == "" {
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}

	db, err := connectDB()
	if err != nil {
		log.Println("Ошибка подключения к БД:", err)
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM profile_photos WHERE photo_url = ?)", req.PhotoURL).Scan(&exists)
	if err != nil {
		log.Println("Ошибка проверки фото в БД:", err)
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	if !exists {
		log.Println("Фото не найдено в базе:", req.PhotoURL)
		http.Error(w, "Фото не найдено", http.StatusNotFound)
		return
	}

	err = deleteFileFromBackblaze(req.PhotoURL)
	if err != nil {
		http.Error(w, "Ошибка удаления фото из хранилища", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("DELETE FROM profile_photos WHERE photo_url = ?", req.PhotoURL)
	if err != nil {
		log.Println("Ошибка удаления из БД:", err)
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	log.Println("Фото успешно удалено:", req.PhotoURL)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"success"}`))
}

func deleteVideoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	type RequestData struct {
		VideoURL string `json:"video"` 
	}

	var req RequestData

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.VideoURL == "" {
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}
	log.Println("🌐 Получен запрос на удаление:", req.VideoURL)

	db, err := connectDB() 
	if err != nil {
		log.Println("Ошибка подключения к БД:", err)
		http.Error(w, "Ошибка подключения к базе данных", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM profile_videos WHERE video_url = ?)", req.VideoURL).Scan(&exists)
	if err != nil {
		log.Println("Ошибка проверки видео в БД:", err)
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	if !exists {
		log.Println("Видео не найдено в базе:", req.VideoURL)
		http.Error(w, "Видео не найдено", http.StatusNotFound)
		return
	}

	err = deleteFileFromBackblaze(req.VideoURL)
	if err != nil {
		http.Error(w, "Ошибка удаления видео из хранилища", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("DELETE FROM profile_videos WHERE video_url = ?", req.VideoURL)
	if err != nil {
		log.Println("Ошибка удаления из БД:", err)
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	log.Println("Видео успешно удалено:", req.VideoURL)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"success"}`))
}

func uploadPhotoHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("📤 Запрос на загрузку фото")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Метод не разрешён"}`, http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(100 << 20) // 10MB
	if err != nil {
		log.Println("Ошибка обработки формы:", err)
		http.Error(w, `{"error": "Ошибка обработки формы"}`, http.StatusBadRequest)
		return
	}

	emails, exists := r.MultipartForm.Value["email"]
	if !exists || len(emails) == 0 {
		log.Println("Ошибка: Email не передан в запросе")
		http.Error(w, `{"error": "Email обязателен"}`, http.StatusBadRequest)
		return
	}
	userEmail := emails[0]
	log.Println("📧 Email пользователя:", userEmail)

	userExists := checkUserExists(userEmail)
	if !userExists {
		log.Println("Ошибка: Пользователь не найден:", userEmail)
		http.Error(w, `{"error": "Пользователь не найден"}`, http.StatusNotFound)
		return
	}

	file, handler, err := r.FormFile("photo")
	if err != nil {
		log.Println("Ошибка получения файла:", err)
		http.Error(w, "Ошибка получения файла", http.StatusBadRequest)
		return
	}
	defer file.Close()
	log.Println("Файл получен:", handler.Filename)

	const maxFileSize = 5 << 20 // 5MB
	if handler.Size > maxFileSize {
		log.Println("Ошибка: Файл слишком большой:", handler.Size)
		http.Error(w, "Файл слишком большой (максимум 5MB)", http.StatusBadRequest)
		return
	}

	allowedMimeTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
	}

	fileHeader := make([]byte, 512)
	if _, err := file.Read(fileHeader); err != nil {
		log.Println("Ошибка чтения заголовка файла:", err)
		http.Error(w, "Ошибка чтения файла", http.StatusBadRequest)
		return
	}

	fileType := http.DetectContentType(fileHeader)
	if !allowedMimeTypes[fileType] {
		log.Println("Ошибка: недопустимый тип файла:", fileType)
		http.Error(w, "Разрешены только изображения (JPEG, PNG, GIF)", http.StatusBadRequest)
		return
	}

	file.Seek(0, 0) 

	fileExt := filepath.Ext(handler.Filename)
	uniqueFileName := uuid.New().String() + fileExt

	fileURL, err := uploadFileToBackblaze(file, uniqueFileName) 
	if err != nil {
		log.Println("Ошибка загрузки в Backblaze:", err)
		http.Error(w, "Ошибка загрузки файла", http.StatusInternalServerError)
		return
	}

	db, err := connectDB()
	if err != nil {
		log.Println("Ошибка подключения к БД:", err)
		http.Error(w, "Ошибка подключения к БД", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var userID int
	err = db.QueryRow("SELECT id FROM profiles WHERE email = ?", userEmail).Scan(&userID)
	if err != nil {
		log.Println("Ошибка: Пользователь не найден:", userEmail)
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	var photoExists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM profile_photos WHERE profile_id = ? AND photo_url = ?)", userID, fileURL).Scan(&photoExists)
	if err != nil {
		log.Println("Ошибка при проверке существования фото в БД:", err)
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	if photoExists {
		log.Println("Файл уже существует в базе:", fileURL)
		http.Error(w, "Такое фото уже загружено", http.StatusConflict)
		return
	}

	_, err = db.Exec("INSERT INTO profile_photos (profile_id, photo_url) VALUES (?, ?)", userID, fileURL)
	if err != nil {
		log.Println("Ошибка записи в БД:", err)
		http.Error(w, "Ошибка сохранения в БД", http.StatusInternalServerError)
		return
	}

	log.Println("Фото загружено:", fileURL)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "photo_url": fileURL})
}

func uploadVideoHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Запрос на загрузку видео")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Метод не разрешён"}`, http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(200 << 20) // 200MB — общий лимит формы
	if err != nil {
		log.Println("Ошибка обработки формы:", err)
		http.Error(w, `{"error": "Ошибка обработки формы"}`, http.StatusBadRequest)
		return
	}

	emails, exists := r.MultipartForm.Value["email"]
	if !exists || len(emails) == 0 {
		log.Println("Email не передан")
		http.Error(w, `{"error": "Email обязателен"}`, http.StatusBadRequest)
		return
	}
	userEmail := emails[0]
	log.Println("Email пользователя:", userEmail)

	if !checkUserExists(userEmail) {
		log.Println("Пользователь не найден:", userEmail)
		http.Error(w, `{"error": "Пользователь не найден"}`, http.StatusNotFound)
		return
	}

	file, handler, err := r.FormFile("video")
	if err != nil {
		log.Println("Ошибка получения видеофайла:", err)
		http.Error(w, "Ошибка получения файла", http.StatusBadRequest)
		return
	}
	defer file.Close()
	log.Println("Видео получено:", handler.Filename)

	const maxVideoSize = 50 << 20 // 50MB
	if handler.Size > maxVideoSize {
		log.Println("Видео слишком большое:", handler.Size)
		http.Error(w, "Максимальный размер видео — 50MB", http.StatusBadRequest)
		return
	}

	allowedVideoTypes := map[string]bool{
		"video/mp4":       true,
		"video/quicktime": true,
		"video/webm":      true,
	}

	fileHeader := make([]byte, 512)
	if _, err := file.Read(fileHeader); err != nil {
		log.Println("Ошибка чтения заголовка файла:", err)
		http.Error(w, "Ошибка чтения файла", http.StatusBadRequest)
		return
	}

	fileType := http.DetectContentType(fileHeader)
	if !allowedVideoTypes[fileType] {
		log.Println("Недопустимый формат видео:", fileType)
		http.Error(w, "Разрешены только MP4, MOV, WEBM", http.StatusBadRequest)
		return
	}

	file.Seek(0, 0) 

	fileExt := filepath.Ext(handler.Filename)
	uniqueFileName := uuid.New().String() + fileExt

	fileURL, err := uploadFileToBackblaze(file, uniqueFileName)
	if err != nil {
		log.Println("Ошибка загрузки в Backblaze:", err)
		http.Error(w, "Ошибка загрузки видео", http.StatusInternalServerError)
		return
	}

	db, err := connectDB()
	if err != nil {
		log.Println("Ошибка подключения к БД:", err)
		http.Error(w, "Ошибка подключения к БД", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var userID int
	err = db.QueryRow("SELECT id FROM profiles WHERE email = ?", userEmail).Scan(&userID)
	if err != nil {
		log.Println("Пользователь не найден при поиске ID:", userEmail)
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	var videoExists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM profile_videos WHERE profile_id = ? AND video_url = ?)", userID, fileURL).Scan(&videoExists)
	if err != nil {
		log.Println("Ошибка при проверке видео в БД:", err)
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	if videoExists {
		log.Println("Видео уже загружено:", fileURL)
		http.Error(w, "Это видео уже загружено", http.StatusConflict)
		return
	}

	_, err = db.Exec("INSERT INTO profile_videos (profile_id, video_url) VALUES (?, ?)", userID, fileURL)
	if err != nil {
		log.Println("Ошибка записи видео в БД:", err)
		http.Error(w, "Ошибка сохранения в базе", http.StatusInternalServerError)
		return
	}

	log.Println("Видео успешно загружено:", fileURL)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "video_url": fileURL})
}

func parseNullableInt(value string) sql.NullInt32 {
	if value == "" {
		return sql.NullInt32{Valid: false}
	}
	parsedValue, err := strconv.Atoi(value)
	if err != nil {
		log.Println("Ошибка конвертации:", value, "Ошибка:", err)
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: int32(parsedValue), Valid: true}
}

func updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Получен запрос на обновление профиля")

	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}
	log.Println("👤 IP клиента:", ip)

	if r.Method != http.MethodPost {
		log.Println("Ошибка: Метод не разрешён")
		http.Error(w, `{"error": "Метод не разрешён"}`, http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Username        string            `json:"username"`
		Password        string            `json:"password"`
		Email           string            `json:"email"`
		ProfileName     string            `json:"profile_name"`
		Phone           string            `json:"phone"`
		Country         string            `json:"country"`
		City            string            `json:"city"`
		District        string            `json:"district"`
		Age             *int              `json:"age"`
		Height          *int              `json:"height"`
		Weight          *int              `json:"weight"`
		EyeColor        string            `json:"eye_color"`
		HairColor       string            `json:"hair_color"`
		HairLength      string            `json:"hair_length"`
		BreastSize      string            `json:"breast_size"`
		BreastType      string            `json:"breast_type"`
		Orientation     string            `json:"orientation"`
		Smoke           string            `json:"smoke"`
		Tattoo          string            `json:"tattoo"`
		Piercing        string            `json:"piercing"`
		Intim           string            `json:"intim"`
		Nationality     string            `json:"nationality"`
		Languages       map[string]string `json:"languages"`
		Messengers      []string          `json:"messengers"`
		Services        []string          `json:"services"`
		About           string            `json:"about"`
		Incall          bool              `json:"incall"`
		Outcall         bool              `json:"outcall"`
		Currency        string            `json:"currency"`
		PriceIncall1h   *int              `json:"price_incall_1h"`
		PriceIncall2h   *int              `json:"price_incall_2h"`
		PriceIncall24h  *int              `json:"price_incall_24h"`
		PriceOutcall1h  *int              `json:"price_outcall_1h"`
		PriceOutcall2h  *int              `json:"price_outcall_2h"`
		PriceOutcall24h *int              `json:"price_outcall_24h"`
	}

	enToRuHairColor := map[string]string{
		"blonde":   "Блондинка",
		"brunette": "Брюнетка",
		"brown":    "Шатенка",
		"red":      "Рыжая",
		"other":    "Другой",
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	log.Println("Тело запроса:", string(bodyBytes))
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Println("Ошибка разбора JSON:", err)
		http.Error(w, `{"error": "Неверный формат данных"}`, http.StatusBadRequest)
		return
	}

	allowedHairLengths := map[string]bool{
		"Короткие": true,
		"Средние":  true,
		"Длинные":  true,
	}

	if val, ok := enToRuHairColor[data.HairColor]; ok {
		data.HairColor = val
		log.Println("Hair color переведён:", val)
	}

	if data.Age == nil || *data.Age < 18 {
		log.Println("Ошибка: возраст меньше 18 или отсутствует")
		http.Error(w, `{"error": "Возраст должен быть не меньше 18 лет"}`, http.StatusBadRequest)
		return
	}


	log.Println("Получено значение hair_length:", data.HairLength)

	var hairLength interface{}
	if data.HairLength != "" {
		if allowedHairLengths[data.HairLength] {
			hairLength = data.HairLength
			log.Println("Принято значение hair_length:", hairLength) 
		} else {
			log.Println("Неверное значение для hair_length:", data.HairLength)
			http.Error(w, `{"error": "Неверное значение длины волос"}`, http.StatusBadRequest)
			return
		}
	} else {
		hairLength = nil
		log.Println("ℹДлина волос не указана, будет записано NULL")
	}

	
	allowedHairColors := map[string]bool{
		"Блондинка": true,
		"Брюнетка":  true,
		"Шатенка":   true,
		"Рыжая":     true,
		"Другой":    true, 
	}

	hairColor := ""
	if data.HairColor != "" {
		if allowedHairColors[data.HairColor] {
			hairColor = data.HairColor
		} else {
			log.Println("Неверное значение для hair_color:", data.HairColor)
			http.Error(w, `{"error": "Неверное значение цвета волос"}`, http.StatusBadRequest)
			return
		}
	}

	rusToDbSmoke := map[string]int{
		"Да":  1,
		"Нет": 0,
	}

	var smoke interface{}
	if data.Smoke != "" {
		if val, ok := rusToDbSmoke[data.Smoke]; ok {
			smoke = val
			log.Println("Курение:", val)
		} else {
			log.Println("Неверное значение smoke:", data.Smoke)
			http.Error(w, `{"error": "Неверное значение курения"}`, http.StatusBadRequest)
			return
		}
	} else {
		smoke = nil
		log.Println("ℹКурение не указано, будет записано NULL")
	}

	rusToDbTattoo := map[string]int{
		"Да":  1,
		"Нет": 0,
	}

	var tattoo interface{}
	if data.Tattoo != "" {
		if val, ok := rusToDbTattoo[data.Tattoo]; ok {
			tattoo = val
			log.Println("Тату (int):", val)
		} else {
			log.Println("Неверное значение tattoo:", data.Tattoo)
			http.Error(w, `{"error": "Неверное значение татуировок"}`, http.StatusBadRequest)
			return
		}
	} else {
		tattoo = nil
		log.Println("Тату не указано, будет записано NULL")
	}

	rusToDbPiercing := map[string]int{
		"Да":  1,
		"Нет": 0,
	}

	var piercing interface{}
	if data.Piercing != "" {
		if val, ok := rusToDbPiercing[data.Piercing]; ok {
			piercing = val
			log.Println("Пирсинг (int):", val)
		} else {
			log.Println("Неверное значение piercing:", data.Piercing)
			http.Error(w, `{"error": "Неверное значение пирсинга"}`, http.StatusBadRequest)
			return
		}
	} else {
		piercing = nil
		log.Println("ℹПирсинг не указан, будет записано NULL")
	}

	rusToDbCountry := map[string]string{
		"Грузия":      "Georgia",
		"Georgia":     "Georgia",
		"Армения":     "Armenia",
		"Armenia":     "Armenia",
		"Азербайджан": "Azerbaijan",
		"Azerbaijan":  "Azerbaijan",
		"Турция":      "Turkey",
		"Turkey":      "Turkey",
		"ОАЭ":         "UAE",
		"UAE":         "UAE",
	}

	var country interface{}
	if data.Country != "" {
		if val, ok := rusToDbCountry[data.Country]; ok {
			country = val
			log.Println("Страна принята:", val)
		} else {
			log.Println("Неверное значение country:", data.Country)
			http.Error(w, `{"error": "Неверное значение страны"}`, http.StatusBadRequest)
			return
		}
	} else {
		country = nil
		log.Println("ℹСтрана не указана, будет записано NULL")
	}

	var height sql.NullInt64
	if data.Height != nil {
		height.Valid = true
		height.Int64 = int64(*data.Height)
	} else {
		height.Valid = false
	}

	var weight sql.NullInt64
	if data.Weight != nil {
		weight.Valid = true
		weight.Int64 = int64(*data.Weight)
	} else {
		weight.Valid = false
	}

	breastType := data.BreastType
	orientation := data.Orientation
	intim := data.Intim

	log.Println("Email пользователя:", data.Email)

	db, err := connectDB()
	if err != nil {
		log.Println("Ошибка подключения к БД:", err)
		http.Error(w, `{"error": "Ошибка базы данных"}`, http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var userID int
	err = db.QueryRow("SELECT id FROM profiles WHERE email = ?", data.Email).Scan(&userID)
	if err == sql.ErrNoRows {
		log.Println("Ошибка: Пользователь не найден:", data.Email)
		http.Error(w, `{"error": "Пользователь не найден"}`, http.StatusNotFound)
		return
	} else if err != nil {
		log.Println("Ошибка запроса к БД:", err)
		http.Error(w, `{"error": "Ошибка базы данных"}`, http.StatusInternalServerError)
		return
	}
	log.Println("Найден пользователь ID:", userID)

	languagesJSON, _ := json.Marshal(data.Languages)

	messengersStr := strings.Join(data.Messengers, ",")
	log.Println("Сохраняем мессенджеры:", messengersStr)

	query := `
                UPDATE profiles
                SET profile_name = ?, phone = ?, country = ?, city = ?, district = ?, age = ?, nationality = ?,
                        height = ?, weight = ?, price_incall_1h = ?, price_incall_2h = ?, price_incall_24h = ?,
                        price_outcall_1h = ?, price_outcall_2h = ?, price_outcall_24h = ?,
                        about = ?, messenger = ?, currency = ?, eye_color = ?, hair_color = ?, hair_length = ?, breast_size = ?,
                        breast_type = ?, orientation = ?, smoker = ?, tattoo = ?, piercing = ?, intim = ?,
                        languages = ?, incall = ?, outcall = ?
                WHERE id = ?
        `

	log.Printf("Отправка данных в базу: eyeColor=%s, hairColor=%s, hairLength=%v, breastType=%s",
		data.EyeColor, hairColor, hairLength, breastType)

	log.Printf("📤 Проверка типов: smoke=%v (%T), tattoo=%v (%T), piercing=%v (%T)", smoke, smoke, tattoo, tattoo, piercing, piercing)

	_, err = db.Exec(query,
		data.ProfileName, data.Phone, country, data.City, data.District, data.Age, data.Nationality,
		height, weight,
		data.PriceIncall1h, data.PriceIncall2h, data.PriceIncall24h,
		data.PriceOutcall1h, data.PriceOutcall2h, data.PriceOutcall24h,
		data.About, messengersStr, data.Currency, data.EyeColor, hairColor, hairLength, data.BreastSize,
		breastType, orientation, smoke, tattoo, piercing, intim,
		string(languagesJSON),
		data.Incall, data.Outcall,
		userID,
	)

	if err != nil {
		log.Println("Ошибка обновления профиля:", err)
		http.Error(w, `{"error": "Ошибка обновления профиля"}`, http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("DELETE FROM services WHERE profile_id = ?", userID)
	if err != nil {
		log.Println("Ошибка при удалении старых услуг:", err)
		http.Error(w, `{"error": "Ошибка при обновлении услуг"}`, http.StatusInternalServerError)
		return
	}

	for _, service := range data.Services {
		_, err := db.Exec("INSERT INTO services (profile_id, service_name, included) VALUES (?, ?, 1)", userID, service)
		if err != nil {
			log.Println("Ошибка при добавлении услуги:", service, "Ошибка:", err)
			http.Error(w, `{"error": "Ошибка при добавлении услуги"}`, http.StatusInternalServerError)
			return
		}
	}

	log.Println("Услуги успешно обновлены")

	log.Println("Профиль успешно обновлён:", data.Email)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status": "success", "message": "Профиль обновлён"}`)
}

func checkEnvVariables() {
	envVars := []string{"DB_USER", "DB_PASS", "DB_HOST", "DB_NAME", "PORT"} 
	for _, env := range envVars {
		value := os.Getenv(env)
		if value == "" {
			log.Fatalf("Ошибка: Переменная окружения %s не задана!", env)
		} else {
			log.Printf("%s загружена", env)
		}
	}
}

func checkUserExists(email string) bool {
	db, err := connectDB()
	if err != nil {
		log.Println("Ошибка подключения к БД:", err)
		return false
	}
	defer db.Close()

	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM profiles WHERE email = ?)", email).Scan(&exists)
	if err != nil {
		log.Println("Ошибка проверки существования пользователя:", err)
		return false
	}
	return exists
}

func getProfileHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Получен запрос GET /api/get-profile")

	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}
	log.Println("IP клиента:", ip)

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		http.Error(w, `{"status":"error","message":"Email обязателен"}`, http.StatusBadRequest)
		return
	}
	email := req.Email

	db, err := connectDB()
	if err != nil {
		http.Error(w, `{"status":"error","message":"Ошибка подключения к БД"}`, http.StatusInternalServerError)
		return
	}
	defer db.Close()
	log.Println("Подключение к базе данных установлено")

	_, err = db.Exec("UPDATE profiles SET last_active = NOW() WHERE email = ?", email)
	if err != nil {
		log.Println("Ошибка при обновлении last_active:", err)
	}

	query := `
SELECT id, username, email, password_hash, profile_name, phone,
       age, height, weight, country, city, district, nationality, body_type,
       languages, about, price_30min, price_1h, price_2h, price_24h,
       messenger, features, created_at, meeting_format, incall, outcall,
       price_12h, eye_color, hair_color, hair_length, breast_size,
       breast_type, orientation, smoker, tattoo, piercing, currency,
       price_incall_1h, price_incall_2h, price_incall_24h,
       price_outcall_1h, price_outcall_2h, price_outcall_24h,
       status, top_status, intim,
       views_total, views_today, views_today_date, verified, last_active
FROM profiles
WHERE email = ?
`

	row := db.QueryRow(query, email)

	var p struct {
		ID              int
		Username        string
		Email           string
		PasswordHash    string
		ProfileName     string
		Phone           string
		Age             sql.NullInt64
		Height          sql.NullInt64
		Weight          sql.NullInt64
		Country         sql.NullString `json:"country"`
		City            string
		District        string
		Nationality     string
		BodyType        sql.NullString
		Languages       string
		About           string
		Price30min      sql.NullInt64
		Price1h         sql.NullInt64
		Price2h         sql.NullInt64
		Price24h        sql.NullInt64
		Messenger       sql.NullString
		Features        sql.NullString
		CreatedAt       sql.NullString
		MeetingFormat   sql.NullString
		Incall          bool
		Outcall         bool
		Price12h        sql.NullInt64
		EyeColor        string
		HairColor       string
		HairLength      sql.NullString
		BreastSize      string
		BreastType      string
		Orientation     string
		Smoker          bool
		Tattoo          bool
		Piercing        bool
		Verified        bool
		Currency        string
		PriceIncall1h   sql.NullInt64
		PriceIncall2h   sql.NullInt64
		PriceIncall24h  sql.NullInt64
		PriceOutcall1h  sql.NullInt64
		PriceOutcall2h  sql.NullInt64
		PriceOutcall24h sql.NullInt64
		Status          string
		TopStatus       bool
		Intim           sql.NullString
		ViewsTotal      int
		ViewsToday      int
		ViewsTodayDate  sql.NullString
		Online          bool
		LastActive      sql.NullString
	}

	err = row.Scan(
		&p.ID, &p.Username, &p.Email, &p.PasswordHash, &p.ProfileName, &p.Phone,
		&p.Age, &p.Height, &p.Weight, &p.Country, &p.City, &p.District, &p.Nationality, &p.BodyType,
		&p.Languages, &p.About, &p.Price30min, &p.Price1h, &p.Price2h, &p.Price24h,
		&p.Messenger, &p.Features, &p.CreatedAt, &p.MeetingFormat, &p.Incall, &p.Outcall,
		&p.Price12h, &p.EyeColor, &p.HairColor, &p.HairLength, &p.BreastSize,
		&p.BreastType, &p.Orientation, &p.Smoker, &p.Tattoo, &p.Piercing, &p.Currency,
		&p.PriceIncall1h, &p.PriceIncall2h, &p.PriceIncall24h,
		&p.PriceOutcall1h, &p.PriceOutcall2h, &p.PriceOutcall24h,
		&p.Status, &p.TopStatus, &p.Intim,
		&p.ViewsTotal, &p.ViewsToday, &p.ViewsTodayDate, &p.Verified, &p.LastActive,
	)

	if err != nil {
		log.Println("Ошибка при выполнении запроса:", err)
		http.Error(w, `{"status":"error","message":"Профиль не найден"}`, http.StatusNotFound)
		return
	}

	serviceRows, err := db.Query("SELECT service_name FROM services WHERE profile_id = ? AND included = 1", p.ID)
	if err != nil {
		log.Println("Ошибка при загрузке услуг:", err)
		http.Error(w, `{"status":"error","message":"Ошибка загрузки услуг"}`, http.StatusInternalServerError)
		return
	}
	defer serviceRows.Close()

	var services []string
	for serviceRows.Next() {
		var name string
		if err := serviceRows.Scan(&name); err == nil {
			services = append(services, name)
		}
	}

	var langs map[string]string
	if err := json.Unmarshal([]byte(p.Languages), &langs); err != nil {
		langs = map[string]string{
			"georgian":    "",
			"russian":     "",
			"english":     "",
			"azerbaijani": "",
		}
	}

	photoRows, err := db.Query("SELECT photo_url FROM profile_photos WHERE profile_id = ?", p.ID)
	if err != nil {
		log.Printf("Ошибка при загрузке фото для profile_id %d: %v\n", p.ID, err)
		http.Error(w, fmt.Sprintf(`{"status":"error","message":"Ошибка загрузки фото: %v"}`, err), http.StatusInternalServerError)
		return
	}

	defer photoRows.Close()

	var photos []string
	for photoRows.Next() {
		var url string
		if err := photoRows.Scan(&url); err == nil {
			photos = append(photos, url)
		}
	}

	videoRows, err := db.Query("SELECT video_url FROM profile_videos WHERE profile_id = ?", p.ID)
	if err != nil {
		log.Println("Ошибка при загрузке видео:", err)
		http.Error(w, `{"status":"error","message":"Ошибка загрузки видео"}`, http.StatusInternalServerError)
		return
	}
	defer videoRows.Close()

	var videos []string
	for videoRows.Next() {
		var url string
		if err := videoRows.Scan(&url); err == nil {
			videos = append(videos, url)
		}
	}

	response := map[string]interface{}{
		"status": "success",
		"profile": map[string]interface{}{
			"username": p.Username,
			"email":    p.Email,
			"profile_name": func() string {
				if p.ProfileName != "" {
					return p.ProfileName
				}
				return ""
			}(),
			"phone": p.Phone,
			"country": func() string {
				if p.Country.Valid {
					return p.Country.String
				}
				return ""
			}(),
			"city":        p.City,
			"district":    p.District,
			"nationality": p.Nationality,
			"age": func() int64 {
				if p.Age.Valid {
					return p.Age.Int64
				}
				return 0
			}(),

			"height": func() int64 {
				if p.Height.Valid {
					return p.Height.Int64
				}
				return 0
			}(),
			"weight": func() int64 {
				if p.Weight.Valid {
					return p.Weight.Int64
				}
				return 0
			}(),
			"eye_color":  p.EyeColor,
			"hair_color": p.HairColor,
			"hair_length": func() string {
				if p.HairLength.Valid {
					return p.HairLength.String
				}
				return ""
			}(),
			"breast_size": p.BreastSize,
			"breast_type": p.BreastType,
			"orientation": p.Orientation,
			"smoke":       p.Smoker,
			"tattoo":      p.Tattoo,
			"piercing":    p.Piercing,
			"about":       p.About,
			"intim": func() string {
				if p.Intim.Valid {
					return p.Intim.String
				}
				return ""
			}(),

			"views_total": p.ViewsTotal,
			"views_today": p.ViewsToday,
			"views_today_date": func() string {
				if p.ViewsTodayDate.Valid {
					return p.ViewsTodayDate.String
				}
				return ""
			}(),

			"last_active": func() string {
				if p.LastActive.Valid {
					return p.LastActive.String
				}
				return ""
			}(),

			"currency": p.Currency,
			"price_30min": func() int64 {
				if p.Price30min.Valid {
					return p.Price30min.Int64
				}
				return 0
			}(),
			"price_1h": func() int64 {
				if p.Price1h.Valid {
					return p.Price1h.Int64
				}
				return 0
			}(),
			"price_2h": func() int64 {
				if p.Price2h.Valid {
					return p.Price2h.Int64
				}
				return 0
			}(),
			"price_24h": func() int64 {
				if p.Price24h.Valid {
					return p.Price24h.Int64
				}
				return 0
			}(),
			"price_12h": func() int64 {
				if p.Price12h.Valid {
					return p.Price12h.Int64
				}
				return 0
			}(),
			"price_incall_1h": func() int64 {
				if p.PriceIncall1h.Valid {
					return p.PriceIncall1h.Int64
				}
				return 0
			}(),
			"price_incall_2h": func() int64 {
				if p.PriceIncall2h.Valid {
					return p.PriceIncall2h.Int64
				}
				return 0
			}(),
			"price_incall_24h": func() int64 {
				if p.PriceIncall24h.Valid {
					return p.PriceIncall24h.Int64
				}
				return 0
			}(),
			"price_outcall_1h": func() int64 {
				if p.PriceOutcall1h.Valid {
					return p.PriceOutcall1h.Int64
				}
				return 0
			}(),
			"price_outcall_2h": func() int64 {
				if p.PriceOutcall2h.Valid {
					return p.PriceOutcall2h.Int64
				}
				return 0
			}(),
			"price_outcall_24h": func() int64 {
				if p.PriceOutcall24h.Valid {
					return p.PriceOutcall24h.Int64
				}
				return 0
			}(),
			"languages": langs,
			"messengers": func() []string {
				if p.Messenger.Valid && p.Messenger.String != "" {
					parts := strings.Split(p.Messenger.String, ",")
					for i := range parts {
						parts[i] = strings.ToLower(strings.TrimSpace(parts[i]))
					}
					return parts
				}
				return []string{}
			}(),
			"services":   services,
			"incall":     p.Incall,
			"outcall":    p.Outcall,
			"status":     p.Status,
			"top_status": p.TopStatus,
			"photos":     photos,
			"videos":     videos,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Println("Профиль отправлен клиенту")
}

func getAllServicesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Получен запрос GET /api/get-services")

	db, err := connectDB()
	if err != nil {
		log.Println("Ошибка подключения к БД:", err)
		http.Error(w, `{"status":"error","message":"DB error"}`, http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT DISTINCT service_name FROM services ORDER BY service_name ASC")
	if err != nil {
		log.Println("Ошибка при запросе услуг:", err)
		http.Error(w, `{"status":"error","message":"Query error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var services []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			services = append(services, name)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
	log.Println("Список всех услуг отправлен клиенту")
}

func toggleStatusHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Получен запрос на /api/toggle-status")

	var req struct {
		Email    string `json:"email"`
		IsActive bool   `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		http.Error(w, `{"status":"error","message":"Неверный формат запроса"}`, http.StatusBadRequest)
		return
	}

	db, err := connectDB()
	if err != nil {
		log.Println("Ошибка подключения к БД:", err)
		http.Error(w, `{"status":"error","message":"Ошибка подключения к БД"}`, http.StatusInternalServerError)
		return
	}
	defer db.Close()

	newStatus := "Hold"
	if req.IsActive {
		newStatus = "Active"
	}

	_, err = db.Exec("UPDATE profiles SET status = ? WHERE email = ?", newStatus, req.Email)
	if err != nil {
		log.Println("Ошибка обновления статуса:", err)
		http.Error(w, `{"status":"error","message":"Ошибка обновления статуса"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("Статус профиля %s успешно обновлён на %s\n", req.Email, newStatus)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

func luxQueensHandler(w http.ResponseWriter, r *http.Request) {
	type LuxQueen struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		Age      int    `json:"age"`
		City     string `json:"city"`
		PhotoURL string `json:"photo_url"`
	}

	country := r.URL.Query().Get("country")
	log.Println("🌍 Запрашиваем Lux Queens для страны:", country)

	db, err := connectDB()
	if err != nil {
		log.Println("Ошибка подключения к базе:", err)
		http.Error(w, "Ошибка подключения к базе данных", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query(`
    SELECT p.id, p.profile_name, p.age, p.city,
        (SELECT photo_url FROM profile_photos WHERE profile_id = p.id LIMIT 1) as photo
    FROM profiles p
    INNER JOIN (
        SELECT profile_id, MAX(active_until) as max_until
        FROM profile_payments
        WHERE frozen = 0 AND active_until >= CURDATE()
        GROUP BY profile_id
    ) pay ON p.id = pay.profile_id
    WHERE p.status = 'Lux Queen' AND p.active = 1 AND p.country = ?
    ORDER BY (p.id = 78) DESC, pay.max_until DESC
    LIMIT 25
`, country)

	if err != nil {
		http.Error(w, "Ошибка запроса", http.StatusInternalServerError)
		log.Println("DB query error:", err)
		return
	}
	defer rows.Close()

	var queens []LuxQueen

	for rows.Next() {
		var q LuxQueen
		var photo sql.NullString

		if err := rows.Scan(&q.ID, &q.Name, &q.Age, &q.City, &photo); err != nil {
			log.Println("Ошибка сканирования:", err)
			continue
		}

		if photo.Valid {
			q.PhotoURL = photo.String
		} else {
			q.PhotoURL = "/static/images/default.jpg"
		}

		queens = append(queens, q)
	}

	if queens == nil {
		queens = []LuxQueen{}
	}

	log.Printf("Найдено Lux Queens: %d\n", len(queens))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(queens)
}

func toggleFreezeHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Запрос на /api/toggle-freeze")

	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email  string `json:"email"`
		Frozen bool   `json:"frozen"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Println("Ошибка декодирования JSON:", err)
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	db, err := connectDB() 
	if err != nil {
		log.Println("Ошибка подключения к БД:", err)
		http.Error(w, "Ошибка подключения к базе данных", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("UPDATE profiles SET frozen = ? WHERE email = ?", req.Frozen, req.Email)
	if err != nil {
		log.Println("Ошибка при обновлении frozen:", err)
		http.Error(w, "Ошибка обновления профиля", http.StatusInternalServerError)
		return
	}

	log.Printf("frozen для %s обновлён на %v\n", req.Email, req.Frozen)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func viewProfileHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("👁 Получен запрос на просмотр анкеты")

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		http.Error(w, `{"status":"error","message":"Email обязателен"}`, http.StatusBadRequest)
		return
	}

	db, err := connectDB()
	if err != nil {
		http.Error(w, `{"status":"error","message":"Ошибка подключения к БД"}`, http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var total, today int
	var lastDate sql.NullString

	err = db.QueryRow(`
        SELECT views_total, views_today, views_today_date
        FROM profiles WHERE email = ?`, req.Email).Scan(&total, &today, &lastDate)
	if err != nil {
		http.Error(w, `{"status":"error","message":"Профиль не найден"}`, http.StatusNotFound)
		return
	}

	currentDate := time.Now().Format("2006-01-02")
	if !lastDate.Valid || lastDate.String != currentDate {
		today = 1
	} else {
		today++
	}
	total++

	_, err = db.Exec(`
        UPDATE profiles
        SET views_total = ?, views_today = ?, views_today_date = ?
        WHERE email = ?`, total, today, currentDate, req.Email)
	if err != nil {
		http.Error(w, `{"status":"error","message":"Ошибка обновления просмотров"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("👁 Просмотры обновлены: total=%d, today=%d", total, today)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"success"}`))
}

func incrementViewsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("👁️ Получен запрос на увеличение просмотров")

	if r.Method != http.MethodPost {
		http.Error(w, `{"status":"error","message":"Метод не разрешён"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProfileID int `json:"profile_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ProfileID == 0 {
		http.Error(w, `{"status":"error","message":"Неверный ID профиля"}`, http.StatusBadRequest)
		return
	}

	db, err := connectDB()
	if err != nil {
		http.Error(w, `{"status":"error","message":"Ошибка базы данных"}`, http.StatusInternalServerError)
		return
	}
	defer db.Close()

	today := time.Now().Format("2006-01-02")

	var lastDate sql.NullString
	var viewsToday int
	err = db.QueryRow(`
        SELECT views_today_date, views_today
        FROM profiles
        WHERE id = ?`, req.ProfileID).Scan(&lastDate, &viewsToday)

	if err != nil {
		log.Println("Ошибка получения данных просмотров:", err)
		http.Error(w, `{"status":"error","message":"Профиль не найден"}`, http.StatusNotFound)
		return
	}

	var newTodayCount int
	if lastDate.Valid && lastDate.String == today {
		newTodayCount = viewsToday + 1
	} else {
		newTodayCount = 1
	}

	_, err = db.Exec(`
        UPDATE profiles
        SET views_total = views_total + 1,
            views_today = ?,
            views_today_date = ?
        WHERE id = ?`,
		newTodayCount, today, req.ProfileID)

	if err != nil {
		log.Println("Ошибка обновления просмотров:", err)
		http.Error(w, `{"status":"error","message":"Ошибка обновления просмотров"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("Просмотры обновлены: id=%d, today=%d, date=%s\n", req.ProfileID, newTodayCount, today)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"success"}`))
}

func FormatCurrencySymbol(code string) string {
	switch code {
	case "USD":
		return "$"
	case "EUR":
		return "€"
	case "GEL":
		return "₾"
	case "RUB":
		return "₽"
	case "TRY":
		return "₺"
	case "AED":
		return "د.إ"
	default:
		return code
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Пользователь вышел из системы")

	http.SetCookie(w, &http.Cookie{
		Name:   "session_token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	emailCookie, err := r.Cookie("user_email")
	if err == nil {
		db, err := connectDB()
		if err == nil {
			defer db.Close()
			_, err = db.Exec("UPDATE profiles SET last_active = NOW() WHERE email = ?", emailCookie.Value)
			if err != nil {
				log.Println("Не удалось обновить last_active при выходе:", err)
			} else {
				log.Println("🕒 last_active обновлено на момент logout для:", emailCookie.Value)
			}
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		http.Error(w, `{"status":"error","message":"Нужен email"}`, http.StatusBadRequest)
		return
	}

	db, err := connectDB()
	if err != nil {
		http.Error(w, `{"status":"error","message":"Ошибка базы данных"}`, http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("UPDATE profiles SET last_active = NOW() WHERE email = ?", req.Email)
	if err != nil {
		http.Error(w, `{"status":"error","message":"Не удалось обновить last_active"}`, http.StatusInternalServerError)
		return
	}

	log.Println("Обновлено поле last_active для email:", req.Email)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"success"}`)
}

func sitemapHandler(w http.ResponseWriter, r *http.Request) {

	log.Println("sitemapHandler вызван")
	db, err := connectDB()
	if err != nil {
		log.Println("Ошибка подключения к БД:", err)
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id FROM profiles WHERE TRIM(UPPER(status)) IN ('ACTIVE', 'LUX QUEEN')`)
	if err != nil {
		log.Println("Ошибка запроса к БД:", err)
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	log.Println("Запрос на sitemap выполнен, начинаем считывание результатов")

	type URL struct {
		Loc string `xml:"loc"`
	}

	type UrlSet struct {
		XMLName xml.Name `xml:"urlset"`
		Xmlns   string   `xml:"xmlns,attr"`
		URLs    []URL    `xml:"url"`
	}

	urls := []URL{
		{Loc: "https://luxegirlsclub.com/"},
		{Loc: "https://luxegirlsclub.com/profiles"},
	}

	seoPages := []string{
		"https://luxegirlsclub.com/escort-tbilisi.html",
		"https://luxegirlsclub.com/escort-batumi.html",
		"https://luxegirlsclub.com/escort-yerevan.html",
		"https://luxegirlsclub.com/escort-baku.html",
		"https://luxegirlsclub.com/escort-istanbul.html",
		"https://luxegirlsclub.com/escort-ankara.html",
		"https://luxegirlsclub.com/escort-dubai.html",
	}

	for _, page := range seoPages {
		urls = append(urls, URL{Loc: page})
	}

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			log.Println("🔹 Найден ID анкеты:", id)
			urls = append(urls, URL{Loc: fmt.Sprintf("https://luxegirlsclub.com/profile/%d", id)})
		} else {
			log.Println("Ошибка чтения ID:", err)
		}
	}

	urlSet := UrlSet{
		Xmlns: "https://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	w.Header().Set("Content-Type", "application/xml")
	xmlData, err := xml.MarshalIndent(urlSet, "", "  ")
	if err != nil {
		log.Println("Ошибка генерации XML:", err)
		http.Error(w, "Ошибка генерации", http.StatusInternalServerError)
		return
	}

	w.Write([]byte(xml.Header))
	w.Write(xmlData)
}

func setMainPhotoHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Запрос на /api/set-main-photo")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Метод не разрешён"}`, http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Email    string `json:"email"`
		PhotoURL string `json:"photo_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		log.Println("Ошибка разбора JSON:", err)
		http.Error(w, `{"error":"Неверный формат данных"}`, http.StatusBadRequest)
		return
	}

	db, err := connectDB()
	if err != nil {
		log.Println("Ошибка подключения к БД:", err)
		http.Error(w, `{"error":"Ошибка базы данных"}`, http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var id int
	err = db.QueryRow("SELECT id FROM profiles WHERE email = ?", data.Email).Scan(&id)
	if err != nil {
		log.Println("Профиль не найден:", data.Email)
		http.Error(w, `{"error":"Профиль не найден"}`, http.StatusNotFound)
		return
	}

	_, err = db.Exec("UPDATE profiles SET main_photo_url = ? WHERE id = ?", data.PhotoURL, id)
	if err != nil {
		log.Println("Ошибка обновления main_photo_url:", err)
		http.Error(w, `{"error":"Не удалось обновить главное фото"}`, http.StatusInternalServerError)
		return
	}

	log.Println("Главное фото обновлено:", data.PhotoURL)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"success"}`)
}

func profileUpHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Запрос на поднятие анкеты")

	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Email string `json:"email"`
	}

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil || payload.Email == "" {
		http.Error(w, "Некорректный email", http.StatusBadRequest)
		return
	}

	db, err := connectDB()
	if err != nil {
		log.Println("Ошибка подключения к БД:", err)
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("UPDATE profiles SET up_timestamp = NOW() WHERE email = ?", payload.Email)
	if err != nil {
		log.Println("Ошибка при обновлении up_timestamp:", err)
		http.Error(w, "Ошибка при обновлении", http.StatusInternalServerError)
		return
	}

	log.Println("Анкета поднята:", payload.Email)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}

func logWhatsappClickHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	profileID := r.FormValue("profile_id")
	if profileID == "" {
		http.Error(w, "Profile ID is required", http.StatusBadRequest)
		return
	}
	log.Printf("💚💬💚💬💚💬💚💬💚💬💚💬")
	log.Printf("💚 📱 ЗАФИКСИРОВАН КЛИК ПО WHATSAPP | ID анкеты: %s", profileID)
	log.Printf("💚💬💚💬💚💬💚💬💚💬💚💬")

	w.WriteHeader(http.StatusOK)
}

func logTelegramClickHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	profileID := r.FormValue("profile_id")
	if profileID == "" {
		http.Error(w, "Profile ID is required", http.StatusBadRequest)
		return
	}

	log.Printf("💙📨💙📨💙📨💙📨💙📨💙📨")
	log.Printf("💙 ✈️ ЗАФИКСИРОВАН КЛИК ПО TELEGRAM | ID анкеты: %s", profileID)
	log.Printf("💙📨💙📨💙📨💙📨💙📨💙📨")

	w.WriteHeader(http.StatusOK)
}


func startServer() {

	checkEnvVariables() 

	mux := http.NewServeMux()

	mux.HandleFunc("/register", registerHandler)
	mux.HandleFunc("/profiles", profilesHandler)
	mux.HandleFunc("/myprofile", myProfileHandler)
	mux.HandleFunc("/api/login", LoginHandler)
	mux.HandleFunc("/delete_photo", deletePhotoHandler)
	mux.HandleFunc("/delete_video", deleteVideoHandler)
	mux.HandleFunc("/upload_photo", uploadPhotoHandler)
	mux.HandleFunc("/upload_video", uploadVideoHandler)
	mux.HandleFunc("/api/update-profile", updateProfileHandler)
	mux.HandleFunc("/profile/", ProfileHandler)
	mux.HandleFunc("/admin/update-profile", updateProfileStatusHandler)
	mux.HandleFunc("/adminpanel", adminPageHandler)
	mux.HandleFunc("/api/get-profile", getProfileHandler)
	mux.HandleFunc("/api/get-services", getAllServicesHandler)
	mux.HandleFunc("/api/toggle-status", toggleStatusHandler)
	mux.HandleFunc("/api/lux-queens", luxQueensHandler)
	mux.HandleFunc("/api/toggle-freeze", toggleFreezeHandler)
	mux.HandleFunc("/api/view-profile", viewProfileHandler)
	mux.HandleFunc("/api/increment-views", incrementViewsHandler)
	mux.HandleFunc("/api/set-main-photo", setMainPhotoHandler)
	mux.HandleFunc("/admin/profiles-json", adminProfilesJSONHandler)
	mux.HandleFunc("/logout", logoutHandler)
	mux.HandleFunc("/api/ping", pingHandler)
	mux.HandleFunc("/api/sitemap", sitemapHandler)
	mux.HandleFunc("/api/profile/up", profileUpHandler)
	mux.HandleFunc("/api/log-whatsapp-click", logWhatsappClickHandler)
	mux.HandleFunc("/api/log-telegram-click", logTelegramClickHandler)

	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/api/sitemap", http.StatusMovedPermanently)
	})

	log.Println("🔹 Зарегистрированные маршруты:")
	log.Println("✅ /register")
	log.Println("✅ /profiles")
	log.Println("✅ /myprofile")
	log.Println("✅ /api/login")
	log.Println("✅ /delete_photo")
	log.Println("✅ /upload_photo")
	log.Println("✅ /upload_video")
	log.Println("✅ /update_profile")
	log.Println("✅ /log-whatsapp-click")
	log.Println("📥 Вызван deleteVideoHandler")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" 
	}

	log.Printf("🚀 Сервер запущен на 0.0.0.0:%s\n", port)

	err := http.ListenAndServe("0.0.0.0:"+port, mux)
	if err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

func main() {
	startServer()
}

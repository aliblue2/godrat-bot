package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/aliblue2/godrat-bot/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/supabase-community/supabase-go"
)

func main() {
	// Load environment
	_ = godotenv.Load()

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	supaBaseUrl := os.Getenv("SUPABASE_URL")
	supaBaseKey := os.Getenv("SUPABASE_API_KEY")

	// Initialize bot
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal("Telegram bot init error:", err)
	}
	bot.Debug = true

	// Initialize Supabase
	client, err := supabase.NewClient(supaBaseUrl, supaBaseKey, nil)
	if err != nil {
		log.Fatal("Supabase init error:", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		text := update.Message.Text

		switch text {
		case "/start":
			bot.Send(tgbotapi.NewMessage(chatID,
				"👋 سلام! خوش اومدی.\n\n"+
					"با دستور /addclass می‌تونی کلاس جدید ثبت کنی.\n"+
					"با دستور /findclass هم می‌تونی کلاس‌ها رو جستجو کنی.\n"+
					"با دستور /listclasses می‌تونی همه کلاس‌ها رو ببینی."))

		case "/addclass":
			bot.Send(tgbotapi.NewMessage(chatID,
				"📝 لطفاً اطلاعات کلاس رو به این شکل بفرست:\n\n"+
					"نام درس | نام استاد | لینک گروه | شماره ترم | primary/other\n\n"+
					"مثال:\n"+
					"سیستم‌عامل | دکتر احمدی | https://t.me/os4041 | 4041 | primary"))
			continue

		case "/findclass":
			bot.Send(tgbotapi.NewMessage(chatID,
				"🔍 لطفاً نام کلاس رو بفرست تا جستجو کنم.\n\n"+
					"مثال:\n"+
					"سیستم‌عامل"))
			continue

		case "/listclasses":
			var results []models.Class
			// Fetch all classes
			resp, _, err := client.From("class").Select("id, name, master, link, semester, is_primary", "", false).
				Execute()

			if err != nil {
				log.Printf("Supabase query error: %v", err)
				bot.Send(tgbotapi.NewMessage(chatID, "❌ خطا در دریافت لیست کلاس‌ها. لطفاً دوباره امتحان کنید."))
				continue
			}

			// Log raw response for debugging
			log.Printf("Supabase response: %s", string(resp))

			err = json.Unmarshal(resp, &results)
			if err != nil {
				log.Printf("Unmarshal error: %v", err)
				bot.Send(tgbotapi.NewMessage(chatID, "❌ خطا در پردازش لیست کلاس‌ها. لطفاً دوباره امتحان کنید."))
				continue
			}

			if len(results) == 0 {
				bot.Send(tgbotapi.NewMessage(chatID, "😕 هیچ کلاسی در پایگاه داده وجود ندارد."))
				continue
			}

			// Separate primary and non-primary classes
			var primaryClasses, otherClasses []models.Class
			for _, c := range results {
				if c.IsPrimary {
					primaryClasses = append(primaryClasses, c)
				} else {
					otherClasses = append(otherClasses, c)
				}
			}

			msg := "📚 لیست همه کلاس‌ها:\n\n"

			// Primary classes section
			if len(primaryClasses) > 0 {
				msg += "🔷 کلاس‌های اصلی:\n"
				for _, c := range primaryClasses {
					msg += fmt.Sprintf("🔹 %s | استاد: %s | ترم: %s | لینک: %s\n",
						c.Name, c.Master, c.Semester, c.Link)
				}
			} else {
				msg += "🔷 هیچ کلاس اصلی‌ای یافت نشد.\n"
			}

			// Non-primary classes section
			if len(otherClasses) > 0 {
				msg += "\n🔶 کلاس‌های غیر اصلی:\n"
				for _, c := range otherClasses {
					msg += fmt.Sprintf("🔹 %s | استاد: %s | ترم: %s | لینک: %s\n",
						c.Name, c.Master, c.Semester, c.Link)
				}
			} else {
				msg += "\n🔶 هیچ کلاس غیر اصلی‌ای یافت نشد.\n"
			}

			bot.Send(tgbotapi.NewMessage(chatID, msg))
			continue
		}

		// Handle class addition
		if strings.Count(text, "|") == 4 {
			parts := strings.Split(text, "|")
			if len(parts) != 5 {
				bot.Send(tgbotapi.NewMessage(chatID, "❌ فرمت ورودی اشتباه است. لطفاً دوباره امتحان کنید."))
				continue
			}

			// Validate semester as a number
			semesterStr := strings.TrimSpace(parts[3])
			if _, err := strconv.Atoi(semesterStr); err != nil {
				bot.Send(tgbotapi.NewMessage(chatID, "❌ شماره ترم باید عدد باشد."))
				continue
			}

			classId := uuid.New()
			class := models.Class{
				Id:        classId,
				Name:      strings.TrimSpace(parts[0]),
				Master:    strings.TrimSpace(parts[1]),
				Link:      strings.TrimSpace(parts[2]),
				Semester:  semesterStr, // Store as string
				IsPrimary: strings.ToLower(strings.TrimSpace(parts[4])) == "primary",
			}

			_, _, err := client.From("class").Insert(class, false, "", "", "").Execute()
			if err != nil {
				bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ خطا در ذخیره کلاس: %s", err.Error())))
				continue
			}
			bot.Send(tgbotapi.NewMessage(chatID, "✅ کلاس با موفقیت ذخیره شد!"))
			continue
		}

		// Handle class search
		searchName := strings.TrimSpace(text)
		if searchName != "" && text != "/addclass" && text != "/start" && text != "/findclass" && text != "/listclasses" {
			var results []models.Class
			// Normalize search term for Persian text
			searchName = strings.TrimSpace(searchName)
			if !utf8.ValidString(searchName) {
				bot.Send(tgbotapi.NewMessage(chatID, "❌ نام جستجو معتبر نیست. لطفاً از حروف معتبر استفاده کنید."))
				continue
			}

			// Log the search term for debugging
			log.Printf("Searching for class: %s", searchName)

			// Perform case-insensitive search
			resp, _, err := client.From("class").Select("id, name, master, link, semester, is_primary", "", false).
				Ilike("name", "%"+searchName+"%").
				Execute()

			if err != nil {
				log.Printf("Supabase query error: %v", err)
				bot.Send(tgbotapi.NewMessage(chatID, "❌ خطا در جستجوی کلاس‌ها. لطفاً دوباره امتحان کنید."))
				continue
			}

			// Log raw response for debugging
			log.Printf("Supabase response: %s", string(resp))

			err = json.Unmarshal(resp, &results)
			if err != nil {
				log.Printf("Unmarshal error: %v", err)
				bot.Send(tgbotapi.NewMessage(chatID, "❌ خطا در پردازش نتایج جستجو. لطفاً دوباره امتحان کنید."))
				continue
			}

			if len(results) == 0 {
				bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("😕 هیچ کلاسی با نام '%s' پیدا نشد.", searchName)))
				continue
			}

			msg := "📚 کلاس‌های پیدا شده:\n\n"
			for _, c := range results {
				primaryStatus := "اصلی"
				if !c.IsPrimary {
					primaryStatus = "غیر اصلی"
				}
				msg += fmt.Sprintf("🔹 %s | استاد: %s | ترم: %s | لینک: %s | نوع: %s\n",
					c.Name, c.Master, c.Semester, c.Link, primaryStatus)
			}
			bot.Send(tgbotapi.NewMessage(chatID, msg))
			continue
		}
	}
}

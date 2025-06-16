// 🔍 Получаем cookie по имени
function getCookie(name) {
    let matches = document.cookie.match(new RegExp(
        "(?:^|; )" + name.replace(/([.*+?^${}()|[\]\\])/g, '\\$1') + "=([^;]*)"
    ));
    return matches ? decodeURIComponent(matches[1]) : undefined;
}

// 🔐 Глобально получаем userEmail
const userEmail = localStorage.getItem("userEmail") || getCookie("userEmail");

// ⏱️ Периодически отправляем ping, чтобы держать статус online
if (userEmail) {
    setInterval(() => {
        fetch("/api/ping", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ email: userEmail })
        }).then(res => {
            if (!res.ok) {
                console.warn("⚠️ Не удалось отправить ping");
            }
        }).catch(err => {
            console.error("❌ Ошибка ping:", err);
        });
    }, 60000); // каждые 60 секунд
}

// 📦 Когда страница загружена
document.addEventListener("DOMContentLoaded", async () => {
    if (!userEmail) {
        alert("❌ Email не найден. Пожалуйста, войдите заново.");
        return;
    }

    console.log("📧 Загружаем профиль для email:", userEmail);

    try {
        const profileRes = await fetch("/api/get-profile", {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify({ email: userEmail })
        });

        const profileData = await profileRes.json();
        console.log("📦 Ответ профиля:", profileData);

        if (profileData.status !== "success") {
            alert("Ошибка при загрузке профиля");
            return;
        }

        const p = profileData.profile;

        // ✅ Услуги — загружаем список доступных и отмечаем выбранные
        const serviceRes = await fetch("/api/get-services");
        const serviceList = await serviceRes.json();
        const container = document.getElementById("services-container");
        container.innerHTML = "";

        const selectedServices = (p.services || []).map(s => s.trim());
        serviceList.forEach(service => {
            const label = document.createElement("label");
            label.className = "service-checkbox";
            const checkbox = document.createElement("input");
            checkbox.type = "checkbox";
            checkbox.value = service;
            checkbox.name = "services";
            if (selectedServices.includes(service)) {
                checkbox.checked = true;
            }
            label.appendChild(checkbox);
            label.appendChild(document.createTextNode(service));
            container.appendChild(label);
        });

        // Заполнение основных данных
        fillBasicProfile(p);
        // Заполнение фото и видео
        loadMedia(p);
document.getElementById("newPhotos").addEventListener("change", function () {
document.getElementById("uploadStatus").style.display = "block";
    const files = Array.from(this.files);
    if (!files || files.length === 0) return;

    const allowedTypes = ["image/jpeg", "image/png", "image/gif"];
    const maxSizeMB = 5;
    const maxFiles = 10;

    // ⚠️ Проверка количества фото
    if (files.length > maxFiles) {
        alert(`❌ Можно загрузить максимум ${maxFiles} фото за раз.`);
        return;
    }

    let validFiles = files.filter(file => {
        if (!allowedTypes.includes(file.type)) {
            alert(`❌ Неверный формат файла: ${file.name}`);
            return false;
        }
        if (file.size > maxSizeMB * 1024 * 1024) {
            alert(`❌ ${file.name} превышает ${maxSizeMB}MB`);
            return false;
        }
        return true;
    });

    if (validFiles.length === 0) {
        alert("❌ Нет допустимых файлов для загрузки.");
        return;
    }

    const uploadFile = (file) => {
        const formData = new FormData();
        formData.append("photo", file);
        formData.append("email", userEmail);

        return fetch("/upload_photo", {
            method: "POST",
            body: formData
        })
        .then(res => res.json())
        .then(data => {
            if (data.status !== "success") {
                throw new Error(data.message || "Ошибка загрузки");
            }
        });
    };

    Promise.all(validFiles.map(uploadFile))
        .then(() => {
            return fetch("/api/get-profile", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ email: userEmail })
            });
        })
        .then(res => res.json())
        .then(updated => {
            loadMedia(updated.profile);
            document.getElementById("uploadStatus").style.display = "none"; // ✅ вот здесь!
        })
        .catch(err => {
            console.error("Ошибка при загрузке фото:", err);
            alert("❌ Ошибка при загрузке одного или нескольких фото.");
            document.getElementById("uploadStatus").style.display = "none"; // ✅ и вот здесь — на случай ошибки
        });
});



document.getElementById("newVideos").addEventListener("change", function () {
console.log("🎥 Видео: загрузка началась");
document.getElementById("uploadStatus").style.display = "block";
    const files = Array.from(this.files);
    if (!files || files.length === 0) return;

    const allowedTypes = ["video/mp4", "video/quicktime", "video/webm"];
    const maxSizeMB = 50;
    const maxFiles = 5;

    // ⚠️ Проверка количества видео
    if (files.length > maxFiles) {
        alert(`❌ Можно загрузить максимум ${maxFiles} видео за раз.`);
        return;
    }

    let validFiles = files.filter(file => {
        if (!allowedTypes.includes(file.type)) {
            alert(`❌ Неверный формат видео: ${file.name}`);
            return false;
        }
        if (file.size > maxSizeMB * 1024 * 1024) {
            alert(`❌ ${file.name} превышает ${maxSizeMB}MB`);
            return false;
        }
        return true;
    });

    if (validFiles.length === 0) {
        alert("❌ Нет допустимых видео для загрузки.");
        return;
    }

    const uploadFile = (file) => {
        const formData = new FormData();
        formData.append("video", file);
        formData.append("email", userEmail);

        return fetch("/upload_video", {
            method: "POST",
            body: formData
        })
        .then(res => res.json())
        .then(data => {
            if (data.status !== "success") {
                throw new Error(data.message || "Ошибка загрузки");
            }
        });
    };

    Promise.all(validFiles.map(uploadFile))
        .then(() => {
            return fetch("/api/get-profile", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ email: userEmail })
            });
        })
        .then(res => res.json())
        .then(updated => {
            loadMedia(updated.profile);
        console.log("🎥 Видео: загрузка завершена");
        document.getElementById("uploadStatus").style.display = "none";
        })
        .catch(err => {
            console.error("Ошибка при загрузке видео:", err);
            alert("❌ Ошибка при загрузке одного или нескольких видео.");
        document.getElementById("uploadStatus").style.display = "none";
        });
});



        // Статистика
        document.getElementById("views_today").textContent = p.views_today || "0";
        document.getElementById("views_total").textContent = p.views_total || "0";

        // Статус анкеты
        setupStatusButton(p);

        // Обработчик сохранения
        setupSaveButton(p);

    } catch (err) {
        console.error("Ошибка при загрузке данных:", err);
        alert("❌ Не удалось загрузить данные. Попробуйте позже.");
    }
});

// Функция для заполнения основных данных профиля
function fillBasicProfile(p) {
    document.getElementById("username").value = p.username || "";
    document.getElementById("email").value = p.email || "";
    document.getElementById("profile_name").value = p.profile_name || "";
    document.getElementById("phone").value = p.phone || "";
    document.getElementById("country").value = p.country || "";
    document.getElementById("city").value = p.city || "";
    document.getElementById("district").value = p.district || "";
    document.getElementById("age").value = p.age || "";
    document.getElementById("height").value = p.height || "";
    document.getElementById("weight").value = p.weight || "";
    document.getElementById("eye_color").value = p.eye_color || "";
    document.getElementById("hair_color").value = p.hair_color || "";
    document.getElementById("hair_length").value = p.hair_length || "";
    document.getElementById("breast_size").value = p.breast_size || "";
    document.getElementById("breast_type").value = p.breast_type || "";
    document.getElementById("orientation").value = p.orientation || "";
    document.getElementById("nationality").value = p.nationality || "";
    document.getElementById("intim").value = p.intim || "";

    // 🔘 Булевы чекбоксы
    document.getElementById("smoke").checked = p.smoke === true;
    document.getElementById("tattoo").checked = p.tattoo === true;
    document.getElementById("piercing").checked = p.piercing === true;
    document.getElementById("incall").checked = p.incall === true;
    document.getElementById("outcall").checked = p.outcall === true;

    // 💬 Языки
    document.getElementById("lang_georgian").value = p.languages?.georgian || "";
    document.getElementById("lang_russian").value = p.languages?.russian || "";
    document.getElementById("lang_english").value = p.languages?.english || "";
    document.getElementById("lang_azerbaijani").value = p.languages?.azerbaijani || "";

    // 📖 Описание
    document.getElementById("about").value = p.about || "";

// ✅ Отмечаем мессенджеры
    console.log("📨 Messengers из профиля:", p.messengers);
    if (Array.isArray(p.messengers)) {
        const mess = p.messengers.map(m => m.toLowerCase());
        document.getElementById("messenger_whatsapp").checked = mess.includes("whatsapp");
        document.getElementById("messenger_telegram").checked = mess.includes("telegram");
    }

    // 💰 Цены и валюта
    document.getElementById("currency").value = p.currency || "";
    document.getElementById("price_incall_1h").value = p.price_incall_1h || "";
    document.getElementById("price_incall_2h").value = p.price_incall_2h || "";
    document.getElementById("price_incall_24h").value = p.price_incall_24h || "";
    document.getElementById("price_outcall_1h").value = p.price_outcall_1h || "";
    document.getElementById("price_outcall_2h").value = p.price_outcall_2h || "";
    document.getElementById("price_outcall_24h").value = p.price_outcall_24h || "";

    // 🔐 Если нужно где-то отобразить статус анкеты
    const status = p.status || "Hold";
    const topStatus = p.top_status === true;

    // Пример: если есть элемент <div id="status-badge">
    const badge = document.getElementById("status-badge");
    if (badge) {
        badge.textContent = status === "Active" ? "🟢 Активна" : "⛔ На модерации";
        badge.className = status === "Active" ? "badge-active" : "badge-hold";
    }

    const topCheckbox = document.getElementById("top_status");
    if (topCheckbox) {
        topCheckbox.checked = topStatus;
    }
}


function loadMedia(p) {
       const photoContainer = document.getElementById("profilePhotos");
    photoContainer.innerHTML = "";
    (p.photos || []).forEach(url => {
        const wrapper = document.createElement("div");
        wrapper.className = "media-wrapper";
        const img = document.createElement("img");
        img.src = url;
        img.className = "profile-photo";
// 🔘 Кнопка "Сделать главным"
const mainBtn = document.createElement("button");
mainBtn.textContent = "Сделать главным";
mainBtn.className = "main-photo-btn";
mainBtn.onclick = () => {
    fetch("/api/set-main-photo", {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify({
            email: userEmail,
            photo_url: url
        })
    })
    .then(res => res.json())
    .then(data => {
        if (data.status === "success") {
            alert("✅ Главное фото обновлено!");
            location.reload();
        } else {
            alert("❌ Ошибка: " + data.message);
        }
    })
    .catch(err => {
        console.error("Ошибка при обновлении главного фото:", err);
        alert("❌ Не удалось сохранить главное фото.");
    });
};

// 🌟 Если это главное фото — показываем отметку
if (url === p.main_photo_url) {
    const badge = document.createElement("div");
    badge.textContent = "🌟 Главное фото";
    badge.className = "main-photo-badge";
    wrapper.appendChild(badge);
} else {
    wrapper.appendChild(mainBtn);
}

        const closeBtn = document.createElement("span");
        closeBtn.innerHTML = "&times;";
        closeBtn.className = "close-btn";
        closeBtn.onclick = () => {
            if (!confirm("Удалить это фото?")) return;
            console.log("📤 Отправляем запрос на удаление фото:", url);

            fetch("/delete_photo", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json"
                },
                body: JSON.stringify({ photo: url })
            })
            .then(res => {
                if (!res.ok) throw new Error("Ошибка удаления");
                return res.json();
            })
            .then(data => {
                if (data.status === "success") {
                    setTimeout(() => {
                        fetch("/api/get-profile", {
                            method: "POST",
                            headers: {
                                "Content-Type": "application/json"
                            },
                            body: JSON.stringify({ email: userEmail })
                        })
                        .then(res => res.json())
                        .then(updated => {
                            loadMedia(updated);
                        });
                    }, 50);
                } else {
                    alert("Ошибка при удалении фото");
                }
            })
            .catch(err => {
                console.error("❌ Ошибка при удалении фото:", err);
                alert("Не удалось удалить фото");
            });
        };
        wrapper.appendChild(img);
        wrapper.appendChild(closeBtn);
        photoContainer.appendChild(wrapper);
    });

     const videoContainer = document.getElementById("profileVideos");
    videoContainer.innerHTML = "";
    (p.videos || []).forEach(url => {
        const wrapper = document.createElement("div");
        wrapper.className = "media-wrapper";
        const video = document.createElement("video");
        video.src = url;
        video.controls = true;
        video.className = "profile-video";
        const closeBtn = document.createElement("span");
        closeBtn.innerHTML = "&times;";
        closeBtn.className = "close-btn";
        closeBtn.onclick = () => {
            if (!confirm("Удалить это видео?")) return;
            console.log("📤 Отправляем запрос на удаление видео:", url);

            fetch("/delete_video", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json"
                },
                body: JSON.stringify({ video: url })
            })
            .then(res => {
                if (!res.ok) throw new Error("Ошибка удаления");
                return res.json();
            })
            .then(data => {
                if (data.status === "success") {
                    setTimeout(() => {
                        fetch("/api/get-profile", {
                            method: "POST",
                            headers: {
                                "Content-Type": "application/json"
                            },
                            body: JSON.stringify({ email: userEmail })
                        })
                        .then(res => res.json())
                        .then(updated => {
                            loadMedia(updated);
                        });
                    }, 50);
                } else {
                    alert("Ошибка при удалении видео");
                }
            })
            .catch(err => {
                console.error("❌ Ошибка при удалении видео:", err);
                alert("Не удалось удалить видео");
            });
        };
        wrapper.appendChild(video);
        wrapper.appendChild(closeBtn);
        videoContainer.appendChild(wrapper);
    });
}



function setupStatusButton(p) {
    const statusBtn = document.getElementById("pauseBtn");
    const freezeText = document.getElementById("freeze-status");
    let isFrozen = p.frozen;
    updateStatusButton();

    statusBtn.addEventListener("click", () => {
        const newFrozen = !isFrozen;
        fetch("/api/toggle-freeze", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ frozen: newFrozen, email: p.email })
        })
        .then(res => res.json())
        .then(data => {
            if (data.status === "success") {
                isFrozen = newFrozen;
                updateStatusButton();
                alert("Статус анкеты обновлён");
            } else {
                alert("Ошибка: " + (data.message || "не удалось обновить статус"));
            }
        })
        .catch(err => {
            console.error("Ошибка при отправке:", err);
            alert("Сервер не отвечает");
        });
    });

    function updateStatusButton() {
    const lang = localStorage.getItem("selectedLanguage") || "ru";
    const key = isFrozen ? "unpause" : "pause";

    // Безопасное получение перевода
    const text = window.translations?.[lang]?.[key] || (isFrozen ? "▶ Возобновить анкету" : "⏸ Приостановить анкету");

    if (statusBtn) {
        statusBtn.textContent = text;
        statusBtn.setAttribute("data-translate", key); // для обновления при смене языка
        statusBtn.classList.toggle("paused", isFrozen);
    }

    if (freezeText) {
        let freezeMessages = {
            ru: {
                active: "🟢 Анкета активна и отображается на сайте",
                paused: "🔒 Анкета приостановлена вами и временно скрыта с сайта"
            },
            en: {
                active: "🟢 Your profile is active and visible on the website",
                paused: "🔒 Your profile is paused and hidden from the website"
            },
            ge: {
                active: "🟢 ანკეტა აქტიურია და ნაჩვენებია საიტზე",
                paused: "🔒 ანკეტა შეჩერებულია და დროებით ფარულია საიტზე"
            }
        };

        const msg = isFrozen ? freezeMessages[lang]?.paused : freezeMessages[lang]?.active;
        freezeText.textContent = msg || "";
        freezeText.style.color = isFrozen ? "gray" : "green";
    }
}

}



// Функция для сохранения изменений
function setupSaveButton(p) {
    const saveBtn = document.getElementById("btn-save");
    if (saveBtn) {
        saveBtn.addEventListener("click", async function (event) {
            event.preventDefault();
            // 🔒 Проверка возраста
            const age = parseInt(document.getElementById("age").value) || 0;
            if (age < 18) {
                alert("❌ Возраст должен быть не меньше 18 лет.");
                return;
            }

            // Проверка длины волос
            const hairLengthEl = document.getElementById("hair_length");
            const hairLength = hairLengthEl?.value.trim() || "";
            const allowedHairLengths = ["Короткие", "Средние", "Длинные"];
            if (hairLength !== "" && !allowedHairLengths.includes(hairLength)) {
                alert("❌ Неверное значение для длины волос!");
                return;
            }

            const breastSize = document.getElementById("breast_size").value.trim();
if (!breastSize) {
    alert("❌ Пожалуйста, выберите размер груди.");
    return; // Останавливаем выполнение, если размер не выбран
}


            // Подготовка данных для отправки
            const data = {
                username: document.getElementById("username").value,
                password: document.getElementById("password").value,
                email: document.getElementById("email").value,
                profile_name: document.getElementById("profile_name").value,
                phone: document.getElementById("phone").value,
                country: document.getElementById("country").value || p.country,
                city: document.getElementById("city").value,
                district: document.getElementById("district").value,
                age: parseInt(document.getElementById("age").value) || 0,
                height: parseInt(document.getElementById("height").value) || null,
                weight: parseInt(document.getElementById("weight").value) || null,
                eye_color: document.getElementById("eye_color").value,
                hair_color: document.getElementById("hair_color").value,
                hair_length: hairLength || null,
                breast_size: document.getElementById("breast_size").value,
                breast_type: document.getElementById("breast_type").value,
                orientation: document.getElementById("orientation").value,
                smoke: document.getElementById("smoke").value,
                tattoo: document.getElementById("tattoo").value,
                piercing: document.getElementById("piercing").value,
                intim: document.getElementById("intim").value,
                nationality: document.getElementById("nationality").value,
                languages: {
                    georgian: document.getElementById("lang_georgian").value,
                    russian: document.getElementById("lang_russian").value,
                    english: document.getElementById("lang_english").value,
                    azerbaijani: document.getElementById("lang_azerbaijani").value
                },
                services: Array.from(document.querySelectorAll('#services-container input[type="checkbox"]:checked'))
                    .map(cb => cb.value),
messengers: [
  document.getElementById("messenger_whatsapp").checked ? "WhatsApp" : null,
  document.getElementById("messenger_telegram").checked ? "Telegram" : null
].filter(Boolean),

                about: document.getElementById("about").value,
                incall: document.getElementById("incall").checked,
                outcall: document.getElementById("outcall").checked,
                currency: document.getElementById("currency").value,
                price_incall_1h: parseInt(document.getElementById("price_incall_1h").value) || null,
                price_incall_2h: parseInt(document.getElementById("price_incall_2h").value) || null,
                price_incall_24h: parseInt(document.getElementById("price_incall_24h").value) || null,
                price_outcall_1h: parseInt(document.getElementById("price_outcall_1h").value) || null,
                price_outcall_2h: parseInt(document.getElementById("price_outcall_2h").value) || null,
                price_outcall_24h: parseInt(document.getElementById("price_outcall_24h").value) || null
            };
	        console.log("📤 Отправляем профиль:", data);

            // Отправка данных на сервер
            try {
                const response = await fetch("/api/update-profile", {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json"
                    },
                    body: JSON.stringify(data)
                });

                const result = await response.json();
                if (result.status === "success") {
                    alert("✅ Профиль успешно обновлён!");
                } else {
                    alert("❌ Ошибка при обновлении профиля: " + result.message);
                }
            } catch (error) {
                console.error("❌ Ошибка при сохранении профиля:", error);
                alert("❌ Не удалось сохранить профиль. Попробуйте позже.");
            }
        });
    } else {
        console.warn("❌ Кнопка btn-save не найдена на странице.");
    }
}

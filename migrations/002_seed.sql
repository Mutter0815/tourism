-- Вставка тестовых данных
INSERT INTO users (telegram_id, username, first_name, last_name, role) VALUES
(1111111111, 'tourist_user', 'Иван', 'Турист', 'user'),
(2222222222, 'provider_user', 'Петр', 'Провайдер', 'provider'),
(3333333333, 'support_user', 'Админ', 'Служба', 'support');

-- Добавим несколько локаций
INSERT INTO locations (name, description, category, region, rating, latitude, longitude, provider_id) VALUES
('Гора Фиагдон', 'Живописное горное место для треккинга', 'Природа', 'Северная Осетия', 4.7, 42.9341, 44.5678, 2),
('Замок Алания', 'Старинный замок для экскурсий', 'История', 'Северная Осетия', 4.3, 42.7000, 44.5000, NULL);

-- Добавим фотографию для первой локации (фиктивный file_id для примера)
INSERT INTO location_photos (location_id, file_id) VALUES
(1, 'TEST_FILE_ID_1');

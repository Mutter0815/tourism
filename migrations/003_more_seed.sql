-- Дополнительные данные для локаций и бронирований

-- Новые локации
INSERT INTO locations (name, description, category, region, rating, latitude, longitude, provider_id) VALUES
('Куртатинское ущелье', 'Популярное место для треккинга и осмотра башен', 'Природа', 'Северная Осетия', 4.8, 42.8389, 44.0630, 2),
('Цейское ущелье', 'Горнолыжный курорт и центр альпинизма', 'Природа', 'Северная Осетия', 4.6, 42.7057, 43.8964, 2);

-- Фотографии для новых локаций (примерные FileID)
INSERT INTO location_photos (location_id, file_id) VALUES
(3, 'TEST_FILE_ID_2'),
(4, 'TEST_FILE_ID_3');

-- Примерные заявки на бронирование
INSERT INTO bookings (user_id, location_id, details, status) VALUES
(1, 1, '2025-08-01 — 2025-08-05, 2 человека', 'pending'),
(1, 3, '2025-09-10 — 2025-09-15, 4 человека', 'confirmed');

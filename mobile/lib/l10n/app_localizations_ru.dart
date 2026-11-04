// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Russian (`ru`).
class AppLocalizationsRu extends AppLocalizations {
  AppLocalizationsRu([String locale = 'ru']) : super(locale);

  @override
  String get appTitle => 'ISOTOPE';

  @override
  String get next => 'Далее';

  @override
  String get cancel => 'Отмена';

  @override
  String get save => 'Сохранить';

  @override
  String get close => 'Закрыть';

  @override
  String get continueLabel => 'Продолжить';

  @override
  String get settings => 'Настройки';

  @override
  String get privacy => 'Приватность';

  @override
  String get language => 'Язык';

  @override
  String get languageEnglish => 'Английский';

  @override
  String get languageRussian => 'Русский';

  @override
  String get languageSystem => 'Системный';

  @override
  String get skip => 'Пропустить';

  @override
  String get back => 'Назад';

  @override
  String get yes => 'Да';

  @override
  String get no => 'Нет';

  @override
  String get ok => 'ОК';

  @override
  String get error => 'Ошибка';

  @override
  String get retry => 'Повторить';

  @override
  String get onboardingWelcomeTitle => 'Добро пожаловать в ISOTOPE';

  @override
  String get onboardingWelcomeBody =>
      'Общайтесь без цензуры и слежки.\nСообщения защищены. Никто не видит,\nкто вы и что вы пишете.';

  @override
  String get onboardingCodeReadyTitle => 'Ваш код готов';

  @override
  String get onboardingCodeReadyBody =>
      'Мы создали для вас уникальный код.\nОн нужен, чтобы друзья могли добавить\nвас в контакты.';

  @override
  String get onboardingAddContactTitle => 'Добавьте первый контакт';

  @override
  String get onboardingAddContactBody =>
      'Отсканируйте QR друга или покажите\nсвой — чтобы начать общение.';

  @override
  String get scan => 'Сканировать';

  @override
  String get showMyQR => 'Показать мой QR';

  @override
  String get privacyShowReadReceipts => 'Показывать статус прочтения';

  @override
  String get privacyShowReadReceiptsSub =>
      'Когда включено — вы видите, прочитал ли собеседник ваше сообщение, и он видит ваш статус. Когда выключено — оба скрыты.';

  @override
  String get privacyShowNotificationContent =>
      'Показывать содержимое уведомлений';

  @override
  String get privacyShowNotificationContentSub =>
      'Когда включено — в уведомлении отображается имя контакта и начало сообщения. Когда выключено — только «Новое сообщение».';

  @override
  String get privacyNetworkNotification => 'Уведомление «Сеть активна»';

  @override
  String get privacyNetworkNotificationSub =>
      'Что показывать в постоянном уведомлении, пока приложение в фоне.';

  @override
  String get privacyShowCount => 'Показывать количество';

  @override
  String get privacyShowCountSub => '«3 непрочитанных»';

  @override
  String get privacyShowNames => 'Показывать имена контактов';

  @override
  String get privacyShowNamesSub => '«Иван (2), Пётр (1)»';

  @override
  String get privacyShowFileProgress => 'Показывать прогресс файлов';

  @override
  String get privacyShowFileProgressSub => '«📎 file.pdf 45%» + прогресс-бар';

  @override
  String get privacyAnonymity => 'Анонимность';

  @override
  String get privacySphinx => 'Sphinx-маршрутизация';

  @override
  String get privacySphinxSub =>
      'Сообщения идут через 2 relay + получатель. Relay не видит, кто с кем. Получатель видит отправителя. Требует 2+ доверенных контакта. Файлы — через Sphinx только при включённом Mixing.';

  @override
  String get privacyMixing => 'Mixing';

  @override
  String get privacyMixingOn =>
      'Скрывает тайминги. Текст: +13–15 сек. Файлы: ~2.7 мин на 1 МБ. Тип сообщения виден relay. Полное скрытие — v2.0.4.';

  @override
  String get privacyMixingOff => 'Доступно только при включённом Sphinx.';

  @override
  String get privacyMasking => 'Маскировка';

  @override
  String get privacyMaskingSub =>
      'Dummy-трафик: пустой пакет каждые 60 сек. Размер 64 КБ — всегда (Sphinx). Трафик ×1.4. Тип, объём, паттерн скрыты.';

  @override
  String get privacySeedSection => 'Seed-фраза';

  @override
  String get privacySeedSectionSub =>
      'Восстановление PeerID. Если потеряете устройство — введёте фразу на новом, PeerID вернётся. Контакты — заново (QR).';

  @override
  String get privacySeedNotSet => 'Seed-фраза не задана';

  @override
  String get privacySeedSet => 'Seed-фраза задана';

  @override
  String get privacySeedCreateOrRestore => 'Создать или восстановить';

  @override
  String get privacySeedShowRestoreOther => 'Показать, восстановить из другой';

  @override
  String get privacyFileRequestSection => 'Запрос на приём файлов';

  @override
  String get privacyFileRequestSectionSub =>
      'Перед получением файла — спросить согласие. Можно задать порог: всегда спрашивать, никогда не спрашивать или спрашивать от определённого размера.';

  @override
  String get privacyFileRequestAlways => 'Всегда запрашивать';

  @override
  String get privacyFileRequestFromMB => 'Запрашивать от определённого размера';

  @override
  String get privacyFileRequestNever => 'Никогда не запрашивать';

  @override
  String get seedDialogTitle => 'Seed-фраза';

  @override
  String get seedDialogHasSeed =>
      'Seed-фраза задана. PeerID восстановим при перезапуске.\n\nВосстановить из другой фразы? PeerID изменится, контакты — заново.';

  @override
  String get seedDialogNoSeed =>
      'Seed-фраза не задана. Сейчас используется случайный PeerID (RSA).\n\nСоздать seed-фразу — PeerID станет восстановимым. После перезапуска PeerID изменится, контакты — заново.';

  @override
  String get seedCreate => 'Создать';

  @override
  String get seedShow => 'Показать';

  @override
  String get seedRestore => 'Восстановить';

  @override
  String get seedWriteDownTitle => 'Запишите фразу';

  @override
  String get seedWriteDownWarning =>
      'Потеряешь фразу — потеряешь PeerID и контакты. Никому не показывайте.';

  @override
  String get seedIWroteIt => 'Я записал';

  @override
  String get seedSaved =>
      'Фраза сохранена. Перезапустите приложение, чтобы применить';

  @override
  String get seedSaveFailed =>
      'Не удалось сохранить в Keystore. Запишите фразу вручную и перезапустите';

  @override
  String get seedCreateFailed => 'Не удалось создать фразу';

  @override
  String get seedYourPhraseTitle => 'Ваша seed-фраза';

  @override
  String get seedYourPhraseWarning =>
      'Никому не показывайте. Кто знает фразу — тот получает ваш PeerID.';

  @override
  String get seedRestoreTitle => 'Восстановить из фразы';

  @override
  String get seedRestoreWarning =>
      'Введите 12 или 24 слова через пробел. PeerID изменится, контакты — заново.';

  @override
  String get restartToApply => 'Перезапустите приложение, чтобы применить';

  @override
  String get settingsTitle => 'Настройки';

  @override
  String get settingsPersonal => 'Личные';

  @override
  String get settingsProfile => 'Профиль';

  @override
  String get settingsProfileSub => 'Ваше имя';

  @override
  String get privacySubtitle => 'Статусы, уведомления, анонимность, seed-фраза';

  @override
  String get settingsMessages => 'Сообщения';

  @override
  String get settingsMessagesSub => 'Задержка, время удаления';

  @override
  String get settingsSystem => 'Системные';

  @override
  String get settingsConnection => 'Подключение';

  @override
  String get settingsConnectionSub => 'Адрес подключения';

  @override
  String get settingsManualCode => 'Ввести код контакта';

  @override
  String get settingsManualCodeSub => 'Для продвинутых';

  @override
  String get addContactTooltip => 'Добавить контакт';

  @override
  String get settingsTooltip => 'Настройки';

  @override
  String get journalTooltip => 'Журнал';

  @override
  String yourAddress(String addr) {
    return '📍 Ваш адрес: $addr';
  }

  @override
  String contactsCount(int count) {
    return 'Контактов: $count';
  }

  @override
  String get noContactsYet => 'Пока никого нет';

  @override
  String get noContactsHint =>
      'Добавьте первый контакт —\nи сможете общаться без цензуры и слежки.';

  @override
  String get addContact => 'Добавить контакт';

  @override
  String get contactsLabel => 'Контакты:';

  @override
  String get journalLabel => 'Журнал:';

  @override
  String get addContactTitle => 'Добавить контакт';

  @override
  String get addContactScanTitle => 'Сканировать QR-код';

  @override
  String get addContactScanSub => 'Друг показывает код, вы сканируете';

  @override
  String get addContactShowQRTitle => 'Показать мой QR-код';

  @override
  String get addContactShowQRSub => 'Друг сканирует ваш код';

  @override
  String get addContactNearbyTitle => 'Найти рядом';

  @override
  String get addContactNearbySub => 'Поиск контактов в той же сети';

  @override
  String get myQRTitle => 'Мой QR (контакт)';

  @override
  String get myQRShow => 'Покажите этот QR другу';

  @override
  String get myQRCode => 'Ваш код для добавления';

  @override
  String get copy => 'Копировать';

  @override
  String get copied => 'Код скопирован';

  @override
  String get qrNotReady => 'Код ещё не готов, подождите';

  @override
  String get qrOwnCode => 'Это ваш собственный код';

  @override
  String get qrNotRecognized => 'Не удалось распознать код';

  @override
  String get qrContactVerified => 'Контакт проверен';

  @override
  String get qrContactNotVerified => 'Контакт не проверен';

  @override
  String get connectionTitle => 'Подключение';

  @override
  String get connectionAddressLabel => 'Адрес подключения';

  @override
  String get connectionAddressHint => '/ip4/.../tcp/9001/ws/p2p/Qm...';

  @override
  String get connectionLeaveEmpty => 'Оставьте пустым, если не нужен.';

  @override
  String get manualCodeTitle => 'Ввести код контакта';

  @override
  String get manualCodeLabel => 'Код контакта';

  @override
  String get manualCodeHint => 'Qm... или /ip4/.../p2p/Qm...';

  @override
  String get manualCodeHintSub => 'Попросите друга показать код.';

  @override
  String get connect => 'Подключиться';

  @override
  String get contactActionsOpenChat => 'Открыть чат';

  @override
  String get contactActionsRename => 'Переименовать';

  @override
  String get contactActionsDelete => 'Удалить';

  @override
  String get renameContactTitle => 'Переименовать контакт';

  @override
  String get renameContactHint => 'Имя контакта';

  @override
  String deleteContactTitle(String name) {
    return 'Удалить «$name»?';
  }

  @override
  String deleteContactBody(String name) {
    return 'Контакт и вся переписка будут удалены.\nЕсли $name напишет снова — вы не увидите его сообщения.\nЧтобы вернуть — попросите новый QR.';
  }

  @override
  String get displayNameTitle => 'Как вас представить?';

  @override
  String get displayNameHint => 'Например: Иван, коллега по работе';

  @override
  String get displayNameSub =>
      'Например: Иван, коллега по работе,\nПётр, мы договаривались на ремонт';

  @override
  String get send => 'Отправить';

  @override
  String get batteryTitle => 'Фоновая работа';

  @override
  String get batteryBody =>
      'ISOTOPE — это P2P-сеть. Чтобы сообщения приходили, когда приложение свёрнуто, разрешите работу в фоне.\n\nЕсли системное окно не открылось — зайдите в:\nНастройки → Батарея → Запуск приложений → ISOTOPE\nи включите «Автозапуск» + «Работа в фоне».\n\nИли нажмите «Настройки приложения» ниже.';

  @override
  String get batteryLater => 'Позже';

  @override
  String get batteryAllow => 'Разрешить';

  @override
  String get batteryAppSettings => 'Настройки приложения';

  @override
  String get displayNameDialogTitle => 'Как вас представить?';

  @override
  String get displayNameDialogHint => 'Например: Иван, коллега по работе';

  @override
  String get displayNameDialogSub =>
      'Например: Иван, коллега по работе,\nПётр, мы договаривались на ремонт';

  @override
  String get sendAction => 'Отправить';

  @override
  String get nearbyPermission =>
      'Разрешите Bluetooth и геолокацию, чтобы искать рядом';

  @override
  String get nearbySearching => 'Поиск контактов рядом...';

  @override
  String get nearbyNotFound => 'Нет контактов';

  @override
  String get networkSearching => 'Поиск контактов через сеть...';

  @override
  String networkFound(int count) {
    return 'Контактов: $count';
  }

  @override
  String searchFailed(String error) {
    return 'Ошибка поиска: $error';
  }

  @override
  String connectFailed(String error) {
    return 'Ошибка подключения: $error';
  }

  @override
  String get addressNotFound => 'Адрес не найден';

  @override
  String get contactSearching => 'Поиск контакта...';

  @override
  String contactFoundShort(String id) {
    return 'Подключено к контакту $id';
  }

  @override
  String get contactConnected => 'Подключено к контакту';

  @override
  String get contactGeneric => 'Контакт';

  @override
  String get contactIsotope => 'Контакт ISOTOPE';

  @override
  String get contactNotVerifiedShort => 'Не проверен';

  @override
  String get contactUnavailable => 'Недоступен';

  @override
  String get previewVoice => '🎤 Голосовое сообщение';

  @override
  String get previewPhoto => '📷 Фото';

  @override
  String get previewFile => 'Файл';

  @override
  String previewDraft(String text) {
    return 'Черновик: «$text»';
  }

  @override
  String get requestSentWaiting => 'Запрос отправлен. Ожидание...';

  @override
  String get requestNotConfirmed => 'Не подтверждён. Отправить запрос?';

  @override
  String get contactError => 'Ошибка';

  @override
  String get scanQrTitle => 'Сканировать QR';

  @override
  String get torchOff => 'Выключить фонарик';

  @override
  String get torchOn => 'Включить фонарик';

  @override
  String get scanQrHint => 'Наведите камеру на QR-код';

  @override
  String get processing => 'Обработка...';

  @override
  String get fromFile => 'Из файла';

  @override
  String get qrNotFoundInImage => 'QR-код не найден в изображении';

  @override
  String get profileTitle => 'Профиль';

  @override
  String get profileYourName => 'Ваше имя';

  @override
  String get profileNameHint => 'Например: Иван';

  @override
  String get profileNameSub =>
      'Это представление по умолчанию.\nИспользуется в QR и запросах.\nМожно не задавать. Тогда контакты увидят только ваш код.';

  @override
  String get profileNameSaved => 'Имя сохранено';

  @override
  String get messagesTitle => 'Сообщения';

  @override
  String get messagesDelaySection => 'ЗАДЕРЖКА ОТПРАВКИ';

  @override
  String get messagesDelaySub =>
      'Даёт возможность передумать. Сообщение отправляется через выбранное время.';

  @override
  String get messagesDelayLabel => 'Задержка отправки:';

  @override
  String get messagesDelayNone => 'Без задержки';

  @override
  String messagesDelaySeconds(int n) {
    return '$n секунд';
  }

  @override
  String get messagesTtlSection => 'ВРЕМЯ УДАЛЕНИЯ СООБЩЕНИЙ';

  @override
  String get messagesTtlSub =>
      'Через какое время сообщения удаляются у вас и у собеседника.';

  @override
  String get messagesTtlNever => 'Не удаляются';

  @override
  String get messagesTtlAfter => 'Удалять через:';

  @override
  String get messagesTtl10s => '10 секунд';

  @override
  String get messagesTtl30s => '30 секунд';

  @override
  String get messagesTtl1m => '1 минута';

  @override
  String get messagesTtl5m => '5 минут';

  @override
  String get messagesTtl15m => '15 минут';

  @override
  String get messagesTtl30m => '30 минут';

  @override
  String get messagesTtl1h => '1 час';

  @override
  String get messagesTtl4h => '4 часа';

  @override
  String get messagesTtl24h => '24 часа';

  @override
  String get messagesModeSection => 'РЕЖИМ УДАЛЕНИЯ';

  @override
  String get messagesModeAfterRead => 'После прочтения';

  @override
  String get messagesModeHard => 'Жёсткий';

  @override
  String get messagesModeAfterReadSub =>
      'После прочтения — сообщение удаляется через выбранное время после того, как собеседник его прочитал.\n\nЕсли прочтение неизвестно (собеседник не делится статусом) — автоматически используется жёсткий режим.\n\nЕсли сообщение не прочитано в течение 48 часов — оно удаляется принудительно.';

  @override
  String get messagesModeHardSub =>
      'Жёсткий — сообщение удаляется через выбранное время после отправки, независимо от прочтения.';

  @override
  String get messagesSecureWarning =>
      '⚠ При выборе от 10 секунд до 1 минуты скриншоты чата с такими сообщениями временно запрещены — до их удаления.';

  @override
  String get logTitle => 'Журнал ISOTOPE';

  @override
  String get logSave => 'Сохранить';

  @override
  String get logClear => 'Очистить';

  @override
  String get logRefresh => 'Обновить';

  @override
  String get logCleared => 'Журнал очищен';

  @override
  String get logEmpty => 'Журнал пуст';
}

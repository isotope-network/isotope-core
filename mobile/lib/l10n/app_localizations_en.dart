// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'ISOTOPE';

  @override
  String get next => 'Next';

  @override
  String get cancel => 'Cancel';

  @override
  String get save => 'Save';

  @override
  String get close => 'Close';

  @override
  String get continueLabel => 'Continue';

  @override
  String get settings => 'Settings';

  @override
  String get privacy => 'Privacy';

  @override
  String get language => 'Language';

  @override
  String get languageEnglish => 'English';

  @override
  String get languageRussian => 'Russian';

  @override
  String get languageSystem => 'System';

  @override
  String get skip => 'Skip';

  @override
  String get back => 'Back';

  @override
  String get yes => 'Yes';

  @override
  String get no => 'No';

  @override
  String get ok => 'OK';

  @override
  String get error => 'Error';

  @override
  String get retry => 'Retry';

  @override
  String get onboardingWelcomeTitle => 'Welcome to ISOTOPE';

  @override
  String get onboardingWelcomeBody =>
      'Communicate without censorship or surveillance.\nMessages are protected. No one sees\nwho you are or what you write.';

  @override
  String get onboardingCodeReadyTitle => 'Your code is ready';

  @override
  String get onboardingCodeReadyBody =>
      'We created a unique code for you.\nIt\'s needed so friends can add\nyou to contacts.';

  @override
  String get onboardingAddContactTitle => 'Add your first contact';

  @override
  String get onboardingAddContactBody =>
      'Scan a friend\'s QR or show\nyours — to start chatting.';

  @override
  String get scan => 'Scan';

  @override
  String get showMyQR => 'Show my QR';

  @override
  String get privacyShowReadReceipts => 'Show read receipts';

  @override
  String get privacyShowReadReceiptsSub =>
      'When enabled — you see if the other person has read your message, and they see your status. When disabled — both are hidden.';

  @override
  String get privacyShowNotificationContent => 'Show notification content';

  @override
  String get privacyShowNotificationContentSub =>
      'When enabled — the notification shows the contact\'s name and the beginning of the message. When disabled — only \"New message\".';

  @override
  String get privacyNetworkNotification => 'Network active notification';

  @override
  String get privacyNetworkNotificationSub =>
      'What to show in the persistent notification while the app is in the background.';

  @override
  String get privacyShowCount => 'Show count';

  @override
  String get privacyShowCountSub => '\"3 unread\"';

  @override
  String get privacyShowNames => 'Show contact names';

  @override
  String get privacyShowNamesSub => '\"Ivan (2), Peter (1)\"';

  @override
  String get privacyShowFileProgress => 'Show file progress';

  @override
  String get privacyShowFileProgressSub => '\"📎 file.pdf 45%\" + progress bar';

  @override
  String get privacyAnonymity => 'Anonymity';

  @override
  String get privacySphinx => 'Sphinx routing';

  @override
  String get privacySphinxSub =>
      'Messages go through 2 relays + recipient. Relays don\'t see who talks to whom. The recipient sees the sender. Requires 2+ trusted contacts. Files — through Sphinx only with Mixing enabled.';

  @override
  String get privacyMixing => 'Mixing';

  @override
  String get privacyMixingOn =>
      'Hides timings. Text: +13–15 sec. Files: ~2.7 min per 1 MB. Message type visible to relays. Full hiding — v2.0.4.';

  @override
  String get privacyMixingOff => 'Available only with Sphinx enabled.';

  @override
  String get privacyMasking => 'Masking';

  @override
  String get privacyMaskingSub =>
      'Dummy traffic: an empty packet every 60 sec. Size 64 KB — always (Sphinx). Traffic ×1.4. Type, volume, pattern hidden.';

  @override
  String get privacySeedSection => 'Seed phrase';

  @override
  String get privacySeedSectionSub =>
      'PeerID recovery. If you lose your device — enter the phrase on a new one, PeerID returns. Contacts — re-add (QR).';

  @override
  String get privacySeedNotSet => 'Seed phrase not set';

  @override
  String get privacySeedSet => 'Seed phrase is set';

  @override
  String get privacySeedCreateOrRestore => 'Create or restore';

  @override
  String get privacySeedShowRestoreOther => 'Show, restore from another';

  @override
  String get privacyFileRequestSection => 'File request';

  @override
  String get privacyFileRequestSectionSub =>
      'Before receiving a file — ask for consent. You can set a threshold: always ask, never ask, or ask from a certain size.';

  @override
  String get privacyFileRequestAlways => 'Always ask';

  @override
  String get privacyFileRequestFromMB => 'Ask from a certain size';

  @override
  String get privacyFileRequestNever => 'Never ask';

  @override
  String get seedDialogTitle => 'Seed phrase';

  @override
  String get seedDialogHasSeed =>
      'Seed phrase is set. PeerID will be restored on restart.\n\nRestore from another phrase? PeerID will change, contacts — re-add.';

  @override
  String get seedDialogNoSeed =>
      'Seed phrase is not set. A random PeerID (RSA) is used now.\n\nCreate a seed phrase — PeerID becomes recoverable. After restart PeerID will change, contacts — re-add.';

  @override
  String get seedCreate => 'Create';

  @override
  String get seedShow => 'Show';

  @override
  String get seedRestore => 'Restore';

  @override
  String get seedWriteDownTitle => 'Write down the phrase';

  @override
  String get seedWriteDownWarning =>
      'Lose the phrase — lose PeerID and contacts. Don\'t show to anyone.';

  @override
  String get seedIWroteIt => 'I wrote it down';

  @override
  String get seedSaved => 'Phrase saved. Restart the app to apply';

  @override
  String get seedSaveFailed =>
      'Failed to save to Keystore. Write down the phrase manually and restart';

  @override
  String get seedCreateFailed => 'Failed to create phrase';

  @override
  String get seedYourPhraseTitle => 'Your seed phrase';

  @override
  String get seedYourPhraseWarning =>
      'Don\'t show to anyone. Whoever knows the phrase gets your PeerID.';

  @override
  String get seedRestoreTitle => 'Restore from phrase';

  @override
  String get seedRestoreWarning =>
      'Enter 12 or 24 words separated by spaces. PeerID will change, contacts — re-add.';

  @override
  String get restartToApply => 'Restart the app to apply';

  @override
  String get settingsTitle => 'Settings';

  @override
  String get settingsPersonal => 'Personal';

  @override
  String get settingsProfile => 'Profile';

  @override
  String get settingsProfileSub => 'Your name';

  @override
  String get privacySubtitle =>
      'Statuses, notifications, anonymity, seed phrase';

  @override
  String get settingsMessages => 'Messages';

  @override
  String get settingsMessagesSub => 'Delay, auto-delete time';

  @override
  String get settingsSystem => 'System';

  @override
  String get settingsConnection => 'Connection';

  @override
  String get settingsConnectionSub => 'Bootstrap address';

  @override
  String get settingsManualCode => 'Enter contact code';

  @override
  String get settingsManualCodeSub => 'For advanced users';

  @override
  String get addContactTooltip => 'Add contact';

  @override
  String get settingsTooltip => 'Settings';

  @override
  String get journalTooltip => 'Journal';

  @override
  String yourAddress(String addr) {
    return '📍 Your address: $addr';
  }

  @override
  String contactsCount(int count) {
    return 'Contacts: $count';
  }

  @override
  String get noContactsYet => 'No contacts yet';

  @override
  String get noContactsHint =>
      'Add your first contact —\nand you can communicate without censorship or surveillance.';

  @override
  String get addContact => 'Add contact';

  @override
  String get contactsLabel => 'Contacts:';

  @override
  String get journalLabel => 'Journal:';

  @override
  String get addContactTitle => 'Add contact';

  @override
  String get addContactScanTitle => 'Scan QR code';

  @override
  String get addContactScanSub => 'A friend shows the code, you scan it';

  @override
  String get addContactShowQRTitle => 'Show my QR code';

  @override
  String get addContactShowQRSub => 'A friend scans your code';

  @override
  String get addContactNearbyTitle => 'Find nearby';

  @override
  String get addContactNearbySub => 'Search for contacts on the same network';

  @override
  String get myQRTitle => 'My QR (contact)';

  @override
  String get myQRShow => 'Show this QR to a friend';

  @override
  String get myQRCode => 'Your code for adding';

  @override
  String get copy => 'Copy';

  @override
  String get copied => 'Code copied';

  @override
  String get qrNotReady => 'Code is not ready yet, please wait';

  @override
  String get qrOwnCode => 'This is your own code';

  @override
  String get qrNotRecognized => 'Could not recognize the code';

  @override
  String get qrContactVerified => 'Contact verified';

  @override
  String get qrContactNotVerified => 'Contact not verified';

  @override
  String get connectionTitle => 'Connection';

  @override
  String get connectionAddressLabel => 'Connection address';

  @override
  String get connectionAddressHint => '/ip4/.../tcp/9001/ws/p2p/Qm...';

  @override
  String get connectionLeaveEmpty => 'Leave empty if not needed.';

  @override
  String get manualCodeTitle => 'Enter contact code';

  @override
  String get manualCodeLabel => 'Contact code';

  @override
  String get manualCodeHint => 'Qm... or /ip4/.../p2p/Qm...';

  @override
  String get manualCodeHintSub => 'Ask a friend to show the code.';

  @override
  String get connect => 'Connect';

  @override
  String get contactActionsOpenChat => 'Open chat';

  @override
  String get contactActionsRename => 'Rename';

  @override
  String get contactActionsDelete => 'Delete';

  @override
  String get renameContactTitle => 'Rename contact';

  @override
  String get renameContactHint => 'Contact name';

  @override
  String deleteContactTitle(String name) {
    return 'Delete \"$name\"?';
  }

  @override
  String deleteContactBody(String name) {
    return 'The contact and all chat history will be deleted.\nIf $name writes again — you won\'t see their messages.\nTo restore — ask for a new QR.';
  }

  @override
  String get displayNameTitle => 'How should we introduce you?';

  @override
  String get displayNameHint => 'For example: Ivan, work colleague';

  @override
  String get displayNameSub =>
      'For example: Ivan, work colleague,\nPeter, we agreed on the repair';

  @override
  String get send => 'Send';

  @override
  String get batteryTitle => 'Background work';

  @override
  String get batteryBody =>
      'ISOTOPE is a P2P network. To receive messages when the app is minimized, allow background work.\n\nIf the system window didn\'t open — go to:\nSettings → Battery → App launch → ISOTOPE\nand enable \"Auto-launch\" + \"Background work\".\n\nOr tap \"App settings\" below.';

  @override
  String get batteryLater => 'Later';

  @override
  String get batteryAllow => 'Allow';

  @override
  String get batteryAppSettings => 'App settings';

  @override
  String get displayNameDialogTitle => 'How should we introduce you?';

  @override
  String get displayNameDialogHint => 'For example: Ivan, work colleague';

  @override
  String get displayNameDialogSub =>
      'For example: Ivan, work colleague,\nPeter, we agreed on the repair';

  @override
  String get sendAction => 'Send';

  @override
  String get nearbyPermission =>
      'Allow Bluetooth and location to search nearby';

  @override
  String get nearbySearching => 'Searching for contacts nearby...';

  @override
  String get nearbyNotFound => 'No contacts';

  @override
  String get networkSearching => 'Searching for contacts via network...';

  @override
  String networkFound(int count) {
    return 'Contacts: $count';
  }

  @override
  String searchFailed(String error) {
    return 'Search failed: $error';
  }

  @override
  String connectFailed(String error) {
    return 'Connection failed: $error';
  }

  @override
  String get addressNotFound => 'Address not found';

  @override
  String get contactSearching => 'Searching for contact...';

  @override
  String contactFoundShort(String id) {
    return 'Connected to contact $id';
  }

  @override
  String get contactConnected => 'Connected to contact';

  @override
  String get contactGeneric => 'Contact';

  @override
  String get contactIsotope => 'ISOTOPE contact';

  @override
  String get contactNotVerifiedShort => 'Not verified';

  @override
  String get contactUnavailable => 'Unavailable';

  @override
  String get previewVoice => '🎤 Voice message';

  @override
  String get previewPhoto => '📷 Photo';

  @override
  String get previewFile => 'File';

  @override
  String previewDraft(String text) {
    return 'Draft: \"$text\"';
  }

  @override
  String get requestSentWaiting => 'Request sent. Waiting...';

  @override
  String get requestNotConfirmed => 'Not confirmed. Send request?';

  @override
  String get contactError => 'Error';

  @override
  String get scanQrTitle => 'Scan QR';

  @override
  String get torchOff => 'Turn off flashlight';

  @override
  String get torchOn => 'Turn on flashlight';

  @override
  String get scanQrHint => 'Point the camera at the QR code';

  @override
  String get processing => 'Processing...';

  @override
  String get fromFile => 'From file';

  @override
  String get qrNotFoundInImage => 'QR code not found in the image';

  @override
  String get profileTitle => 'Profile';

  @override
  String get profileYourName => 'Your name';

  @override
  String get profileNameHint => 'For example: Ivan';

  @override
  String get profileNameSub =>
      'This is the default representation.\nUsed in QR and requests.\nCan be left empty. Then contacts will see only your code.';

  @override
  String get profileNameSaved => 'Name saved';

  @override
  String get messagesTitle => 'Messages';

  @override
  String get messagesDelaySection => 'SEND DELAY';

  @override
  String get messagesDelaySub =>
      'Gives a chance to change your mind. The message is sent after the selected time.';

  @override
  String get messagesDelayLabel => 'Send delay:';

  @override
  String get messagesDelayNone => 'No delay';

  @override
  String messagesDelaySeconds(int n) {
    return '$n sec';
  }

  @override
  String get messagesTtlSection => 'MESSAGE AUTO-DELETE TIME';

  @override
  String get messagesTtlSub =>
      'After what time messages are deleted for you and the recipient.';

  @override
  String get messagesTtlNever => 'Never';

  @override
  String get messagesTtlAfter => 'Delete after:';

  @override
  String get messagesTtl10s => '10 seconds';

  @override
  String get messagesTtl30s => '30 seconds';

  @override
  String get messagesTtl1m => '1 minute';

  @override
  String get messagesTtl5m => '5 minutes';

  @override
  String get messagesTtl15m => '15 minutes';

  @override
  String get messagesTtl30m => '30 minutes';

  @override
  String get messagesTtl1h => '1 hour';

  @override
  String get messagesTtl4h => '4 hours';

  @override
  String get messagesTtl24h => '24 hours';

  @override
  String get messagesModeSection => 'DELETE MODE';

  @override
  String get messagesModeAfterRead => 'After reading';

  @override
  String get messagesModeHard => 'Hard';

  @override
  String get messagesModeAfterReadSub =>
      'After reading — the message is deleted after the selected time after the recipient reads it.\n\nIf read status is unknown (the recipient doesn\'t share status) — hard mode is used automatically.\n\nIf the message is not read within 48 hours — it is deleted forcibly.';

  @override
  String get messagesModeHardSub =>
      'Hard — the message is deleted after the selected time after sending, regardless of reading.';

  @override
  String get messagesSecureWarning =>
      '⚠ When selecting from 10 seconds to 1 minute, screenshots of chats with such messages are temporarily disabled — until they are deleted.';

  @override
  String get logTitle => 'ISOTOPE Journal';

  @override
  String get logSave => 'Save';

  @override
  String get logClear => 'Clear';

  @override
  String get logRefresh => 'Refresh';

  @override
  String get logCleared => 'Journal cleared';

  @override
  String get logEmpty => 'Journal is empty';
}

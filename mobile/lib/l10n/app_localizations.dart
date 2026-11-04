import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_en.dart';
import 'app_localizations_ru.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'l10n/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
      : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations)!;
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
    delegate,
    GlobalMaterialLocalizations.delegate,
    GlobalCupertinoLocalizations.delegate,
    GlobalWidgetsLocalizations.delegate,
  ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('en'),
    Locale('ru')
  ];

  /// No description provided for @appTitle.
  ///
  /// In en, this message translates to:
  /// **'ISOTOPE'**
  String get appTitle;

  /// No description provided for @next.
  ///
  /// In en, this message translates to:
  /// **'Next'**
  String get next;

  /// No description provided for @cancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get cancel;

  /// No description provided for @save.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get save;

  /// No description provided for @close.
  ///
  /// In en, this message translates to:
  /// **'Close'**
  String get close;

  /// No description provided for @continueLabel.
  ///
  /// In en, this message translates to:
  /// **'Continue'**
  String get continueLabel;

  /// No description provided for @settings.
  ///
  /// In en, this message translates to:
  /// **'Settings'**
  String get settings;

  /// No description provided for @privacy.
  ///
  /// In en, this message translates to:
  /// **'Privacy'**
  String get privacy;

  /// No description provided for @language.
  ///
  /// In en, this message translates to:
  /// **'Language'**
  String get language;

  /// No description provided for @languageEnglish.
  ///
  /// In en, this message translates to:
  /// **'English'**
  String get languageEnglish;

  /// No description provided for @languageRussian.
  ///
  /// In en, this message translates to:
  /// **'Russian'**
  String get languageRussian;

  /// No description provided for @languageSystem.
  ///
  /// In en, this message translates to:
  /// **'System'**
  String get languageSystem;

  /// No description provided for @skip.
  ///
  /// In en, this message translates to:
  /// **'Skip'**
  String get skip;

  /// No description provided for @back.
  ///
  /// In en, this message translates to:
  /// **'Back'**
  String get back;

  /// No description provided for @yes.
  ///
  /// In en, this message translates to:
  /// **'Yes'**
  String get yes;

  /// No description provided for @no.
  ///
  /// In en, this message translates to:
  /// **'No'**
  String get no;

  /// No description provided for @ok.
  ///
  /// In en, this message translates to:
  /// **'OK'**
  String get ok;

  /// No description provided for @error.
  ///
  /// In en, this message translates to:
  /// **'Error'**
  String get error;

  /// No description provided for @retry.
  ///
  /// In en, this message translates to:
  /// **'Retry'**
  String get retry;

  /// No description provided for @onboardingWelcomeTitle.
  ///
  /// In en, this message translates to:
  /// **'Welcome to ISOTOPE'**
  String get onboardingWelcomeTitle;

  /// No description provided for @onboardingWelcomeBody.
  ///
  /// In en, this message translates to:
  /// **'Communicate without censorship or surveillance.\nMessages are protected. No one sees\nwho you are or what you write.'**
  String get onboardingWelcomeBody;

  /// No description provided for @onboardingCodeReadyTitle.
  ///
  /// In en, this message translates to:
  /// **'Your code is ready'**
  String get onboardingCodeReadyTitle;

  /// No description provided for @onboardingCodeReadyBody.
  ///
  /// In en, this message translates to:
  /// **'We created a unique code for you.\nIt\'s needed so friends can add\nyou to contacts.'**
  String get onboardingCodeReadyBody;

  /// No description provided for @onboardingAddContactTitle.
  ///
  /// In en, this message translates to:
  /// **'Add your first contact'**
  String get onboardingAddContactTitle;

  /// No description provided for @onboardingAddContactBody.
  ///
  /// In en, this message translates to:
  /// **'Scan a friend\'s QR or show\nyours — to start chatting.'**
  String get onboardingAddContactBody;

  /// No description provided for @scan.
  ///
  /// In en, this message translates to:
  /// **'Scan'**
  String get scan;

  /// No description provided for @showMyQR.
  ///
  /// In en, this message translates to:
  /// **'Show my QR'**
  String get showMyQR;

  /// No description provided for @privacyShowReadReceipts.
  ///
  /// In en, this message translates to:
  /// **'Show read receipts'**
  String get privacyShowReadReceipts;

  /// No description provided for @privacyShowReadReceiptsSub.
  ///
  /// In en, this message translates to:
  /// **'When enabled — you see if the other person has read your message, and they see your status. When disabled — both are hidden.'**
  String get privacyShowReadReceiptsSub;

  /// No description provided for @privacyShowNotificationContent.
  ///
  /// In en, this message translates to:
  /// **'Show notification content'**
  String get privacyShowNotificationContent;

  /// No description provided for @privacyShowNotificationContentSub.
  ///
  /// In en, this message translates to:
  /// **'When enabled — the notification shows the contact\'s name and the beginning of the message. When disabled — only \"New message\".'**
  String get privacyShowNotificationContentSub;

  /// No description provided for @privacyNetworkNotification.
  ///
  /// In en, this message translates to:
  /// **'Network active notification'**
  String get privacyNetworkNotification;

  /// No description provided for @privacyNetworkNotificationSub.
  ///
  /// In en, this message translates to:
  /// **'What to show in the persistent notification while the app is in the background.'**
  String get privacyNetworkNotificationSub;

  /// No description provided for @privacyShowCount.
  ///
  /// In en, this message translates to:
  /// **'Show count'**
  String get privacyShowCount;

  /// No description provided for @privacyShowCountSub.
  ///
  /// In en, this message translates to:
  /// **'\"3 unread\"'**
  String get privacyShowCountSub;

  /// No description provided for @privacyShowNames.
  ///
  /// In en, this message translates to:
  /// **'Show contact names'**
  String get privacyShowNames;

  /// No description provided for @privacyShowNamesSub.
  ///
  /// In en, this message translates to:
  /// **'\"Ivan (2), Peter (1)\"'**
  String get privacyShowNamesSub;

  /// No description provided for @privacyShowFileProgress.
  ///
  /// In en, this message translates to:
  /// **'Show file progress'**
  String get privacyShowFileProgress;

  /// No description provided for @privacyShowFileProgressSub.
  ///
  /// In en, this message translates to:
  /// **'\"📎 file.pdf 45%\" + progress bar'**
  String get privacyShowFileProgressSub;

  /// No description provided for @privacyAnonymity.
  ///
  /// In en, this message translates to:
  /// **'Anonymity'**
  String get privacyAnonymity;

  /// No description provided for @privacySphinx.
  ///
  /// In en, this message translates to:
  /// **'Sphinx routing'**
  String get privacySphinx;

  /// No description provided for @privacySphinxSub.
  ///
  /// In en, this message translates to:
  /// **'Messages go through 2 relays + recipient. Relays don\'t see who talks to whom. The recipient sees the sender. Requires 2+ trusted contacts. Files — through Sphinx only with Mixing enabled.'**
  String get privacySphinxSub;

  /// No description provided for @privacyMixing.
  ///
  /// In en, this message translates to:
  /// **'Mixing'**
  String get privacyMixing;

  /// No description provided for @privacyMixingOn.
  ///
  /// In en, this message translates to:
  /// **'Hides timings. Text: +13–15 sec. Files: ~2.7 min per 1 MB. Message type visible to relays. Full hiding — v2.0.4.'**
  String get privacyMixingOn;

  /// No description provided for @privacyMixingOff.
  ///
  /// In en, this message translates to:
  /// **'Available only with Sphinx enabled.'**
  String get privacyMixingOff;

  /// No description provided for @privacyMasking.
  ///
  /// In en, this message translates to:
  /// **'Masking'**
  String get privacyMasking;

  /// No description provided for @privacyMaskingSub.
  ///
  /// In en, this message translates to:
  /// **'Dummy traffic: an empty packet every 60 sec. Size 64 KB — always (Sphinx). Traffic ×1.4. Type, volume, pattern hidden.'**
  String get privacyMaskingSub;

  /// No description provided for @privacySeedSection.
  ///
  /// In en, this message translates to:
  /// **'Seed phrase'**
  String get privacySeedSection;

  /// No description provided for @privacySeedSectionSub.
  ///
  /// In en, this message translates to:
  /// **'PeerID recovery. If you lose your device — enter the phrase on a new one, PeerID returns. Contacts — re-add (QR).'**
  String get privacySeedSectionSub;

  /// No description provided for @privacySeedNotSet.
  ///
  /// In en, this message translates to:
  /// **'Seed phrase not set'**
  String get privacySeedNotSet;

  /// No description provided for @privacySeedSet.
  ///
  /// In en, this message translates to:
  /// **'Seed phrase is set'**
  String get privacySeedSet;

  /// No description provided for @privacySeedCreateOrRestore.
  ///
  /// In en, this message translates to:
  /// **'Create or restore'**
  String get privacySeedCreateOrRestore;

  /// No description provided for @privacySeedShowRestoreOther.
  ///
  /// In en, this message translates to:
  /// **'Show, restore from another'**
  String get privacySeedShowRestoreOther;

  /// No description provided for @privacyFileRequestSection.
  ///
  /// In en, this message translates to:
  /// **'File request'**
  String get privacyFileRequestSection;

  /// No description provided for @privacyFileRequestSectionSub.
  ///
  /// In en, this message translates to:
  /// **'Before receiving a file — ask for consent. You can set a threshold: always ask, never ask, or ask from a certain size.'**
  String get privacyFileRequestSectionSub;

  /// No description provided for @privacyFileRequestAlways.
  ///
  /// In en, this message translates to:
  /// **'Always ask'**
  String get privacyFileRequestAlways;

  /// No description provided for @privacyFileRequestFromMB.
  ///
  /// In en, this message translates to:
  /// **'Ask from a certain size'**
  String get privacyFileRequestFromMB;

  /// No description provided for @privacyFileRequestNever.
  ///
  /// In en, this message translates to:
  /// **'Never ask'**
  String get privacyFileRequestNever;

  /// No description provided for @seedDialogTitle.
  ///
  /// In en, this message translates to:
  /// **'Seed phrase'**
  String get seedDialogTitle;

  /// No description provided for @seedDialogHasSeed.
  ///
  /// In en, this message translates to:
  /// **'Seed phrase is set. PeerID will be restored on restart.\n\nRestore from another phrase? PeerID will change, contacts — re-add.'**
  String get seedDialogHasSeed;

  /// No description provided for @seedDialogNoSeed.
  ///
  /// In en, this message translates to:
  /// **'Seed phrase is not set. A random PeerID (RSA) is used now.\n\nCreate a seed phrase — PeerID becomes recoverable. After restart PeerID will change, contacts — re-add.'**
  String get seedDialogNoSeed;

  /// No description provided for @seedCreate.
  ///
  /// In en, this message translates to:
  /// **'Create'**
  String get seedCreate;

  /// No description provided for @seedShow.
  ///
  /// In en, this message translates to:
  /// **'Show'**
  String get seedShow;

  /// No description provided for @seedRestore.
  ///
  /// In en, this message translates to:
  /// **'Restore'**
  String get seedRestore;

  /// No description provided for @seedWriteDownTitle.
  ///
  /// In en, this message translates to:
  /// **'Write down the phrase'**
  String get seedWriteDownTitle;

  /// No description provided for @seedWriteDownWarning.
  ///
  /// In en, this message translates to:
  /// **'Lose the phrase — lose PeerID and contacts. Don\'t show to anyone.'**
  String get seedWriteDownWarning;

  /// No description provided for @seedIWroteIt.
  ///
  /// In en, this message translates to:
  /// **'I wrote it down'**
  String get seedIWroteIt;

  /// No description provided for @seedSaved.
  ///
  /// In en, this message translates to:
  /// **'Phrase saved. Restart the app to apply'**
  String get seedSaved;

  /// No description provided for @seedSaveFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed to save to Keystore. Write down the phrase manually and restart'**
  String get seedSaveFailed;

  /// No description provided for @seedCreateFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed to create phrase'**
  String get seedCreateFailed;

  /// No description provided for @seedYourPhraseTitle.
  ///
  /// In en, this message translates to:
  /// **'Your seed phrase'**
  String get seedYourPhraseTitle;

  /// No description provided for @seedYourPhraseWarning.
  ///
  /// In en, this message translates to:
  /// **'Don\'t show to anyone. Whoever knows the phrase gets your PeerID.'**
  String get seedYourPhraseWarning;

  /// No description provided for @seedRestoreTitle.
  ///
  /// In en, this message translates to:
  /// **'Restore from phrase'**
  String get seedRestoreTitle;

  /// No description provided for @seedRestoreWarning.
  ///
  /// In en, this message translates to:
  /// **'Enter 12 or 24 words separated by spaces. PeerID will change, contacts — re-add.'**
  String get seedRestoreWarning;

  /// No description provided for @restartToApply.
  ///
  /// In en, this message translates to:
  /// **'Restart the app to apply'**
  String get restartToApply;

  /// No description provided for @settingsTitle.
  ///
  /// In en, this message translates to:
  /// **'Settings'**
  String get settingsTitle;

  /// No description provided for @settingsPersonal.
  ///
  /// In en, this message translates to:
  /// **'Personal'**
  String get settingsPersonal;

  /// No description provided for @settingsProfile.
  ///
  /// In en, this message translates to:
  /// **'Profile'**
  String get settingsProfile;

  /// No description provided for @settingsProfileSub.
  ///
  /// In en, this message translates to:
  /// **'Your name'**
  String get settingsProfileSub;

  /// No description provided for @privacySubtitle.
  ///
  /// In en, this message translates to:
  /// **'Statuses, notifications, anonymity, seed phrase'**
  String get privacySubtitle;

  /// No description provided for @settingsMessages.
  ///
  /// In en, this message translates to:
  /// **'Messages'**
  String get settingsMessages;

  /// No description provided for @settingsMessagesSub.
  ///
  /// In en, this message translates to:
  /// **'Delay, auto-delete time'**
  String get settingsMessagesSub;

  /// No description provided for @settingsSystem.
  ///
  /// In en, this message translates to:
  /// **'System'**
  String get settingsSystem;

  /// No description provided for @settingsConnection.
  ///
  /// In en, this message translates to:
  /// **'Connection'**
  String get settingsConnection;

  /// No description provided for @settingsConnectionSub.
  ///
  /// In en, this message translates to:
  /// **'Bootstrap address'**
  String get settingsConnectionSub;

  /// No description provided for @settingsManualCode.
  ///
  /// In en, this message translates to:
  /// **'Enter contact code'**
  String get settingsManualCode;

  /// No description provided for @settingsManualCodeSub.
  ///
  /// In en, this message translates to:
  /// **'For advanced users'**
  String get settingsManualCodeSub;

  /// No description provided for @addContactTooltip.
  ///
  /// In en, this message translates to:
  /// **'Add contact'**
  String get addContactTooltip;

  /// No description provided for @settingsTooltip.
  ///
  /// In en, this message translates to:
  /// **'Settings'**
  String get settingsTooltip;

  /// No description provided for @journalTooltip.
  ///
  /// In en, this message translates to:
  /// **'Journal'**
  String get journalTooltip;

  /// No description provided for @yourAddress.
  ///
  /// In en, this message translates to:
  /// **'📍 Your address: {addr}'**
  String yourAddress(String addr);

  /// No description provided for @contactsCount.
  ///
  /// In en, this message translates to:
  /// **'Contacts: {count}'**
  String contactsCount(int count);

  /// No description provided for @noContactsYet.
  ///
  /// In en, this message translates to:
  /// **'No contacts yet'**
  String get noContactsYet;

  /// No description provided for @noContactsHint.
  ///
  /// In en, this message translates to:
  /// **'Add your first contact —\nand you can communicate without censorship or surveillance.'**
  String get noContactsHint;

  /// No description provided for @addContact.
  ///
  /// In en, this message translates to:
  /// **'Add contact'**
  String get addContact;

  /// No description provided for @contactsLabel.
  ///
  /// In en, this message translates to:
  /// **'Contacts:'**
  String get contactsLabel;

  /// No description provided for @journalLabel.
  ///
  /// In en, this message translates to:
  /// **'Journal:'**
  String get journalLabel;

  /// No description provided for @addContactTitle.
  ///
  /// In en, this message translates to:
  /// **'Add contact'**
  String get addContactTitle;

  /// No description provided for @addContactScanTitle.
  ///
  /// In en, this message translates to:
  /// **'Scan QR code'**
  String get addContactScanTitle;

  /// No description provided for @addContactScanSub.
  ///
  /// In en, this message translates to:
  /// **'A friend shows the code, you scan it'**
  String get addContactScanSub;

  /// No description provided for @addContactShowQRTitle.
  ///
  /// In en, this message translates to:
  /// **'Show my QR code'**
  String get addContactShowQRTitle;

  /// No description provided for @addContactShowQRSub.
  ///
  /// In en, this message translates to:
  /// **'A friend scans your code'**
  String get addContactShowQRSub;

  /// No description provided for @addContactNearbyTitle.
  ///
  /// In en, this message translates to:
  /// **'Find nearby'**
  String get addContactNearbyTitle;

  /// No description provided for @addContactNearbySub.
  ///
  /// In en, this message translates to:
  /// **'Search for contacts on the same network'**
  String get addContactNearbySub;

  /// No description provided for @myQRTitle.
  ///
  /// In en, this message translates to:
  /// **'My QR (contact)'**
  String get myQRTitle;

  /// No description provided for @myQRShow.
  ///
  /// In en, this message translates to:
  /// **'Show this QR to a friend'**
  String get myQRShow;

  /// No description provided for @myQRCode.
  ///
  /// In en, this message translates to:
  /// **'Your code for adding'**
  String get myQRCode;

  /// No description provided for @copy.
  ///
  /// In en, this message translates to:
  /// **'Copy'**
  String get copy;

  /// No description provided for @copied.
  ///
  /// In en, this message translates to:
  /// **'Code copied'**
  String get copied;

  /// No description provided for @qrNotReady.
  ///
  /// In en, this message translates to:
  /// **'Code is not ready yet, please wait'**
  String get qrNotReady;

  /// No description provided for @qrOwnCode.
  ///
  /// In en, this message translates to:
  /// **'This is your own code'**
  String get qrOwnCode;

  /// No description provided for @qrNotRecognized.
  ///
  /// In en, this message translates to:
  /// **'Could not recognize the code'**
  String get qrNotRecognized;

  /// No description provided for @qrContactVerified.
  ///
  /// In en, this message translates to:
  /// **'Contact verified'**
  String get qrContactVerified;

  /// No description provided for @qrContactNotVerified.
  ///
  /// In en, this message translates to:
  /// **'Contact not verified'**
  String get qrContactNotVerified;

  /// No description provided for @connectionTitle.
  ///
  /// In en, this message translates to:
  /// **'Connection'**
  String get connectionTitle;

  /// No description provided for @connectionAddressLabel.
  ///
  /// In en, this message translates to:
  /// **'Connection address'**
  String get connectionAddressLabel;

  /// No description provided for @connectionAddressHint.
  ///
  /// In en, this message translates to:
  /// **'/ip4/.../tcp/9001/ws/p2p/Qm...'**
  String get connectionAddressHint;

  /// No description provided for @connectionLeaveEmpty.
  ///
  /// In en, this message translates to:
  /// **'Leave empty if not needed.'**
  String get connectionLeaveEmpty;

  /// No description provided for @manualCodeTitle.
  ///
  /// In en, this message translates to:
  /// **'Enter contact code'**
  String get manualCodeTitle;

  /// No description provided for @manualCodeLabel.
  ///
  /// In en, this message translates to:
  /// **'Contact code'**
  String get manualCodeLabel;

  /// No description provided for @manualCodeHint.
  ///
  /// In en, this message translates to:
  /// **'Qm... or /ip4/.../p2p/Qm...'**
  String get manualCodeHint;

  /// No description provided for @manualCodeHintSub.
  ///
  /// In en, this message translates to:
  /// **'Ask a friend to show the code.'**
  String get manualCodeHintSub;

  /// No description provided for @connect.
  ///
  /// In en, this message translates to:
  /// **'Connect'**
  String get connect;

  /// No description provided for @contactActionsOpenChat.
  ///
  /// In en, this message translates to:
  /// **'Open chat'**
  String get contactActionsOpenChat;

  /// No description provided for @contactActionsRename.
  ///
  /// In en, this message translates to:
  /// **'Rename'**
  String get contactActionsRename;

  /// No description provided for @contactActionsDelete.
  ///
  /// In en, this message translates to:
  /// **'Delete'**
  String get contactActionsDelete;

  /// No description provided for @renameContactTitle.
  ///
  /// In en, this message translates to:
  /// **'Rename contact'**
  String get renameContactTitle;

  /// No description provided for @renameContactHint.
  ///
  /// In en, this message translates to:
  /// **'Contact name'**
  String get renameContactHint;

  /// No description provided for @deleteContactTitle.
  ///
  /// In en, this message translates to:
  /// **'Delete \"{name}\"?'**
  String deleteContactTitle(String name);

  /// No description provided for @deleteContactBody.
  ///
  /// In en, this message translates to:
  /// **'The contact and all chat history will be deleted.\nIf {name} writes again — you won\'t see their messages.\nTo restore — ask for a new QR.'**
  String deleteContactBody(String name);

  /// No description provided for @displayNameTitle.
  ///
  /// In en, this message translates to:
  /// **'How should we introduce you?'**
  String get displayNameTitle;

  /// No description provided for @displayNameHint.
  ///
  /// In en, this message translates to:
  /// **'For example: Ivan, work colleague'**
  String get displayNameHint;

  /// No description provided for @displayNameSub.
  ///
  /// In en, this message translates to:
  /// **'For example: Ivan, work colleague,\nPeter, we agreed on the repair'**
  String get displayNameSub;

  /// No description provided for @send.
  ///
  /// In en, this message translates to:
  /// **'Send'**
  String get send;

  /// No description provided for @batteryTitle.
  ///
  /// In en, this message translates to:
  /// **'Background work'**
  String get batteryTitle;

  /// No description provided for @batteryBody.
  ///
  /// In en, this message translates to:
  /// **'ISOTOPE is a P2P network. To receive messages when the app is minimized, allow background work.\n\nIf the system window didn\'t open — go to:\nSettings → Battery → App launch → ISOTOPE\nand enable \"Auto-launch\" + \"Background work\".\n\nOr tap \"App settings\" below.'**
  String get batteryBody;

  /// No description provided for @batteryLater.
  ///
  /// In en, this message translates to:
  /// **'Later'**
  String get batteryLater;

  /// No description provided for @batteryAllow.
  ///
  /// In en, this message translates to:
  /// **'Allow'**
  String get batteryAllow;

  /// No description provided for @batteryAppSettings.
  ///
  /// In en, this message translates to:
  /// **'App settings'**
  String get batteryAppSettings;

  /// No description provided for @displayNameDialogTitle.
  ///
  /// In en, this message translates to:
  /// **'How should we introduce you?'**
  String get displayNameDialogTitle;

  /// No description provided for @displayNameDialogHint.
  ///
  /// In en, this message translates to:
  /// **'For example: Ivan, work colleague'**
  String get displayNameDialogHint;

  /// No description provided for @displayNameDialogSub.
  ///
  /// In en, this message translates to:
  /// **'For example: Ivan, work colleague,\nPeter, we agreed on the repair'**
  String get displayNameDialogSub;

  /// No description provided for @sendAction.
  ///
  /// In en, this message translates to:
  /// **'Send'**
  String get sendAction;

  /// No description provided for @nearbyPermission.
  ///
  /// In en, this message translates to:
  /// **'Allow Bluetooth and location to search nearby'**
  String get nearbyPermission;

  /// No description provided for @nearbySearching.
  ///
  /// In en, this message translates to:
  /// **'Searching for contacts nearby...'**
  String get nearbySearching;

  /// No description provided for @nearbyNotFound.
  ///
  /// In en, this message translates to:
  /// **'No contacts'**
  String get nearbyNotFound;

  /// No description provided for @networkSearching.
  ///
  /// In en, this message translates to:
  /// **'Searching for contacts via network...'**
  String get networkSearching;

  /// No description provided for @networkFound.
  ///
  /// In en, this message translates to:
  /// **'Contacts: {count}'**
  String networkFound(int count);

  /// No description provided for @searchFailed.
  ///
  /// In en, this message translates to:
  /// **'Search failed: {error}'**
  String searchFailed(String error);

  /// No description provided for @connectFailed.
  ///
  /// In en, this message translates to:
  /// **'Connection failed: {error}'**
  String connectFailed(String error);

  /// No description provided for @addressNotFound.
  ///
  /// In en, this message translates to:
  /// **'Address not found'**
  String get addressNotFound;

  /// No description provided for @contactSearching.
  ///
  /// In en, this message translates to:
  /// **'Searching for contact...'**
  String get contactSearching;

  /// No description provided for @contactFoundShort.
  ///
  /// In en, this message translates to:
  /// **'Connected to contact {id}'**
  String contactFoundShort(String id);

  /// No description provided for @contactConnected.
  ///
  /// In en, this message translates to:
  /// **'Connected to contact'**
  String get contactConnected;

  /// No description provided for @contactGeneric.
  ///
  /// In en, this message translates to:
  /// **'Contact'**
  String get contactGeneric;

  /// No description provided for @contactIsotope.
  ///
  /// In en, this message translates to:
  /// **'ISOTOPE contact'**
  String get contactIsotope;

  /// No description provided for @contactNotVerifiedShort.
  ///
  /// In en, this message translates to:
  /// **'Not verified'**
  String get contactNotVerifiedShort;

  /// No description provided for @contactUnavailable.
  ///
  /// In en, this message translates to:
  /// **'Unavailable'**
  String get contactUnavailable;

  /// No description provided for @previewVoice.
  ///
  /// In en, this message translates to:
  /// **'🎤 Voice message'**
  String get previewVoice;

  /// No description provided for @previewPhoto.
  ///
  /// In en, this message translates to:
  /// **'📷 Photo'**
  String get previewPhoto;

  /// No description provided for @previewFile.
  ///
  /// In en, this message translates to:
  /// **'File'**
  String get previewFile;

  /// No description provided for @previewDraft.
  ///
  /// In en, this message translates to:
  /// **'Draft: \"{text}\"'**
  String previewDraft(String text);

  /// No description provided for @requestSentWaiting.
  ///
  /// In en, this message translates to:
  /// **'Request sent. Waiting...'**
  String get requestSentWaiting;

  /// No description provided for @requestNotConfirmed.
  ///
  /// In en, this message translates to:
  /// **'Not confirmed. Send request?'**
  String get requestNotConfirmed;

  /// No description provided for @contactError.
  ///
  /// In en, this message translates to:
  /// **'Error'**
  String get contactError;

  /// No description provided for @scanQrTitle.
  ///
  /// In en, this message translates to:
  /// **'Scan QR'**
  String get scanQrTitle;

  /// No description provided for @torchOff.
  ///
  /// In en, this message translates to:
  /// **'Turn off flashlight'**
  String get torchOff;

  /// No description provided for @torchOn.
  ///
  /// In en, this message translates to:
  /// **'Turn on flashlight'**
  String get torchOn;

  /// No description provided for @scanQrHint.
  ///
  /// In en, this message translates to:
  /// **'Point the camera at the QR code'**
  String get scanQrHint;

  /// No description provided for @processing.
  ///
  /// In en, this message translates to:
  /// **'Processing...'**
  String get processing;

  /// No description provided for @fromFile.
  ///
  /// In en, this message translates to:
  /// **'From file'**
  String get fromFile;

  /// No description provided for @qrNotFoundInImage.
  ///
  /// In en, this message translates to:
  /// **'QR code not found in the image'**
  String get qrNotFoundInImage;

  /// No description provided for @profileTitle.
  ///
  /// In en, this message translates to:
  /// **'Profile'**
  String get profileTitle;

  /// No description provided for @profileYourName.
  ///
  /// In en, this message translates to:
  /// **'Your name'**
  String get profileYourName;

  /// No description provided for @profileNameHint.
  ///
  /// In en, this message translates to:
  /// **'For example: Ivan'**
  String get profileNameHint;

  /// No description provided for @profileNameSub.
  ///
  /// In en, this message translates to:
  /// **'This is the default representation.\nUsed in QR and requests.\nCan be left empty. Then contacts will see only your code.'**
  String get profileNameSub;

  /// No description provided for @profileNameSaved.
  ///
  /// In en, this message translates to:
  /// **'Name saved'**
  String get profileNameSaved;

  /// No description provided for @messagesTitle.
  ///
  /// In en, this message translates to:
  /// **'Messages'**
  String get messagesTitle;

  /// No description provided for @messagesDelaySection.
  ///
  /// In en, this message translates to:
  /// **'SEND DELAY'**
  String get messagesDelaySection;

  /// No description provided for @messagesDelaySub.
  ///
  /// In en, this message translates to:
  /// **'Gives a chance to change your mind. The message is sent after the selected time.'**
  String get messagesDelaySub;

  /// No description provided for @messagesDelayLabel.
  ///
  /// In en, this message translates to:
  /// **'Send delay:'**
  String get messagesDelayLabel;

  /// No description provided for @messagesDelayNone.
  ///
  /// In en, this message translates to:
  /// **'No delay'**
  String get messagesDelayNone;

  /// No description provided for @messagesDelaySeconds.
  ///
  /// In en, this message translates to:
  /// **'{n} sec'**
  String messagesDelaySeconds(int n);

  /// No description provided for @messagesTtlSection.
  ///
  /// In en, this message translates to:
  /// **'MESSAGE AUTO-DELETE TIME'**
  String get messagesTtlSection;

  /// No description provided for @messagesTtlSub.
  ///
  /// In en, this message translates to:
  /// **'After what time messages are deleted for you and the recipient.'**
  String get messagesTtlSub;

  /// No description provided for @messagesTtlNever.
  ///
  /// In en, this message translates to:
  /// **'Never'**
  String get messagesTtlNever;

  /// No description provided for @messagesTtlAfter.
  ///
  /// In en, this message translates to:
  /// **'Delete after:'**
  String get messagesTtlAfter;

  /// No description provided for @messagesTtl10s.
  ///
  /// In en, this message translates to:
  /// **'10 seconds'**
  String get messagesTtl10s;

  /// No description provided for @messagesTtl30s.
  ///
  /// In en, this message translates to:
  /// **'30 seconds'**
  String get messagesTtl30s;

  /// No description provided for @messagesTtl1m.
  ///
  /// In en, this message translates to:
  /// **'1 minute'**
  String get messagesTtl1m;

  /// No description provided for @messagesTtl5m.
  ///
  /// In en, this message translates to:
  /// **'5 minutes'**
  String get messagesTtl5m;

  /// No description provided for @messagesTtl15m.
  ///
  /// In en, this message translates to:
  /// **'15 minutes'**
  String get messagesTtl15m;

  /// No description provided for @messagesTtl30m.
  ///
  /// In en, this message translates to:
  /// **'30 minutes'**
  String get messagesTtl30m;

  /// No description provided for @messagesTtl1h.
  ///
  /// In en, this message translates to:
  /// **'1 hour'**
  String get messagesTtl1h;

  /// No description provided for @messagesTtl4h.
  ///
  /// In en, this message translates to:
  /// **'4 hours'**
  String get messagesTtl4h;

  /// No description provided for @messagesTtl24h.
  ///
  /// In en, this message translates to:
  /// **'24 hours'**
  String get messagesTtl24h;

  /// No description provided for @messagesModeSection.
  ///
  /// In en, this message translates to:
  /// **'DELETE MODE'**
  String get messagesModeSection;

  /// No description provided for @messagesModeAfterRead.
  ///
  /// In en, this message translates to:
  /// **'After reading'**
  String get messagesModeAfterRead;

  /// No description provided for @messagesModeHard.
  ///
  /// In en, this message translates to:
  /// **'Hard'**
  String get messagesModeHard;

  /// No description provided for @messagesModeAfterReadSub.
  ///
  /// In en, this message translates to:
  /// **'After reading — the message is deleted after the selected time after the recipient reads it.\n\nIf read status is unknown (the recipient doesn\'t share status) — hard mode is used automatically.\n\nIf the message is not read within 48 hours — it is deleted forcibly.'**
  String get messagesModeAfterReadSub;

  /// No description provided for @messagesModeHardSub.
  ///
  /// In en, this message translates to:
  /// **'Hard — the message is deleted after the selected time after sending, regardless of reading.'**
  String get messagesModeHardSub;

  /// No description provided for @messagesSecureWarning.
  ///
  /// In en, this message translates to:
  /// **'⚠ When selecting from 10 seconds to 1 minute, screenshots of chats with such messages are temporarily disabled — until they are deleted.'**
  String get messagesSecureWarning;

  /// No description provided for @logTitle.
  ///
  /// In en, this message translates to:
  /// **'ISOTOPE Journal'**
  String get logTitle;

  /// No description provided for @logSave.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get logSave;

  /// No description provided for @logClear.
  ///
  /// In en, this message translates to:
  /// **'Clear'**
  String get logClear;

  /// No description provided for @logRefresh.
  ///
  /// In en, this message translates to:
  /// **'Refresh'**
  String get logRefresh;

  /// No description provided for @logCleared.
  ///
  /// In en, this message translates to:
  /// **'Journal cleared'**
  String get logCleared;

  /// No description provided for @logEmpty.
  ///
  /// In en, this message translates to:
  /// **'Journal is empty'**
  String get logEmpty;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['en', 'ru'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'en':
      return AppLocalizationsEn();
    case 'ru':
      return AppLocalizationsRu();
  }

  throw FlutterError(
      'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
      'an issue with the localizations generation tool. Please file an issue '
      'on GitHub with a reproducible sample app and the gen-l10n configuration '
      'that was used.');
}

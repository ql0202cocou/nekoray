# macOS platform sources (empty - no platform-specific sources needed)
set(PLATFORM_SOURCES "")

# macOS platform libraries
set(PLATFORM_LIBRARIES "")

# Build a Finder-launchable .app bundle instead of a bare executable.
set(MACOSX_BUNDLE_GUI_IDENTIFIER "com.nekobox.nekobox")
set(MACOSX_BUNDLE_BUNDLE_NAME "nekobox")
set(MACOSX_BUNDLE_DISPLAY_NAME "nekobox")
set(MACOSX_BUNDLE_SHORT_VERSION_STRING "${PROJECT_VERSION}")
set(MACOSX_BUNDLE_BUNDLE_VERSION "${PROJECT_VERSION}")
set(MACOSX_BUNDLE_COPYRIGHT "nekobox")

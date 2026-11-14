import java.util.Properties

plugins { id("com.android.application") }

// clé de signature : keystore.properties (hors git), voir README.md (Applis mobiles)
val keys = Properties().apply { rootProject.file("keystore.properties").takeIf { it.exists() }?.inputStream()?.use { load(it) } }

android {
    namespace = "io.github.lbouzac.patate"
    compileSdk = 36
    defaultConfig {
        applicationId = "io.github.lbouzac.patate"
        minSdk = 26
        targetSdk = 36
        versionCode = 1
        versionName = "1.0"
    }
    // la page d'accueil est partagée avec iOS
    sourceSets.getByName("main").assets.srcDir("../../www")
    signingConfigs {
        if (keys.isNotEmpty()) create("release") {
            storeFile = rootProject.file(keys.getProperty("storeFile"))
            storePassword = keys.getProperty("storePassword")
            keyAlias = keys.getProperty("keyAlias")
            keyPassword = keys.getProperty("keyPassword")
        }
    }
    buildTypes {
        release { signingConfig = signingConfigs.findByName("release") }
    }
}

dependencies {
    implementation("androidx.activity:activity-ktx:1.13.0")
    implementation("androidx.fragment:fragment:1.9.1") // Play services en tire une trop vieille pour registerForActivityResult
    implementation("com.google.android.gms:play-services-code-scanner:16.1.0")
}

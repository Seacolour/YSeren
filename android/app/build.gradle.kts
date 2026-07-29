plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

android {
    namespace = "io.github.seacolour.yseren.mobile"
    compileSdk = 34

    val yserenVersionName = providers.gradleProperty("yserenVersionName")
        .orElse("dev")
        .get()
        .trim()
    val yserenVersionCode = providers.gradleProperty("yserenVersionCode")
        .orElse("1")
        .get()
        .trim()
        .toIntOrNull()
        ?: error("yserenVersionCode must be an integer")
    require(Regex("""^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$""").matches(yserenVersionName) || yserenVersionName == "dev") {
        "yserenVersionName must be dev or MAJOR.MINOR.PATCH, got: $yserenVersionName"
    }
    require(yserenVersionCode > 0) {
        "yserenVersionCode must be positive"
    }

    val androidKeystorePath = System.getenv("ANDROID_KEYSTORE_PATH")
    val androidKeystorePassword = System.getenv("ANDROID_KEYSTORE_PASSWORD")
    val androidKeyAlias = System.getenv("ANDROID_KEY_ALIAS")
    val androidKeyPassword = System.getenv("ANDROID_KEY_PASSWORD")
    val hasReleaseSigning = listOf(
        androidKeystorePath,
        androidKeystorePassword,
        androidKeyAlias,
        androidKeyPassword,
    ).all { !it.isNullOrBlank() }

    defaultConfig {
        applicationId = "io.github.seacolour.yseren.mobile"
        minSdk = 26
        targetSdk = 34
        versionCode = yserenVersionCode
        versionName = yserenVersionName

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    signingConfigs {
        create("release") {
            if (hasReleaseSigning) {
                storeFile = file(androidKeystorePath!!)
                storePassword = androidKeystorePassword
                keyAlias = androidKeyAlias
                keyPassword = androidKeyPassword
            }
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            if (hasReleaseSigning) {
                signingConfig = signingConfigs.getByName("release")
            }
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro",
            )
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }

    buildFeatures {
        compose = true
    }

    sourceSets {
        getByName("main") {
            assets.srcDir(rootProject.file("../frontend/dist"))
        }
        getByName("test") {
            resources.srcDir(rootProject.file("../contracts/fixtures"))
        }
    }

    composeOptions {
        kotlinCompilerExtensionVersion = "1.5.14"
    }
}

dependencies {
    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.appcompat:appcompat:1.7.0")
    implementation("androidx.activity:activity-ktx:1.9.1")
    implementation("androidx.activity:activity-compose:1.9.1")
    implementation(platform("androidx.compose:compose-bom:2024.06.00"))
    implementation("androidx.compose.foundation:foundation")
    implementation("androidx.compose.material:material-icons-extended")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.ui:ui-tooling-preview")
    implementation("androidx.documentfile:documentfile:1.0.1")
    implementation("com.google.android.material:material:1.12.0")
    implementation("org.nanohttpd:nanohttpd:2.3.1")

    debugImplementation("androidx.compose.ui:ui-tooling")
    testImplementation("junit:junit:4.13.2")
    testImplementation("org.json:json:20240303")
}

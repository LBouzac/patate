// swift-tools-version: 6.0
import PackageDescription

// Construit sur Linux avec xtool (xtool dev), sur Mac avec XcodeGen (project.yml) pour TestFlight.
let package = Package(
    name: "Patate",
    platforms: [.iOS(.v17)],
    products: [.library(name: "Patate", targets: ["Patate"])],
    targets: [.target(name: "Patate")]
)

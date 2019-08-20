import UIKit
import Repm

@UIApplicationMain
class AppDelegate: UIResponder, UIApplicationDelegate {

    func application(_ application: UIApplication, didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?) -> Bool {
        // Override point for customization after application launch.
        return true
    }

    // MARK: UISceneSession Lifecycle

    func application(_ application: UIApplication, configurationForConnecting connectingSceneSession: UISceneSession, options: UIScene.ConnectionOptions) -> UISceneConfiguration {
        // Called when a new scene session is being created.
        // Use this method to select a configuration to create the new scene with.
        return UISceneConfiguration(name: "Default Configuration", sessionRole: connectingSceneSession.role)
    }
  
    lazy var replicant: RepmConnection = {
        let appDir = FileManager.default.urls(for: .libraryDirectory, in: .userDomainMask).first!
        let url = appDir.appendingPathComponent("replicant")
        var error: NSError?
        return RepmOpen(url.path, "c1", &error)!
    }()
}

import UIKit
import SwiftUI

class SceneDelegate: UIResponder, UIWindowSceneDelegate {

    var window: UIWindow?

    func scene(_ scene: UIScene, willConnectTo session: UISceneSession, options connectionOptions: UIScene.ConnectionOptions) {
        if let windowScene = scene as? UIWindowScene {
            let replicant = (UIApplication.shared.delegate as? AppDelegate)!.replicant
            let req = ["code": replicantFunctions]
            try! replicant.dispatch("putBundle", data: JSONEncoder().encode(req))
            let rootView = ContentView(replicant: replicant)

            let window = UIWindow(windowScene: windowScene)
            window.rootViewController = UIHostingController(rootView: rootView)
            self.window = window
            window.makeKeyAndVisible()
        }
    }
}

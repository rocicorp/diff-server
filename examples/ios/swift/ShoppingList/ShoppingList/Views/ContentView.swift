import SwiftUI
import CoreData
import Foundation
import Repm

struct ContentView: View {
    
    var replicant: RepmConnection!
        
    @State private var text = ""
    
    @State private var notes = ""

    @State private var isAdding = false {
        didSet {
            if !isAdding {
                self.text = ""
                self.notes = ""
            }
        }
    }
    
    // MARK: Private methods

    private func toggleAdding() {
        isAdding.toggle()
    }
    
    private func save() {
        let req = ["name": "addItem", "args": [text, notes]] as [String : Any]
        let data = try! JSONSerialization.data(withJSONObject: req)
        try! replicant.dispatch("exec", data: data)
        isAdding = false
    }

    // MARK: Render
    
    var body: some View {
        NavigationView {
            Form {
                if isAdding {
                    Section(header: Text("Add an Item")) {
                        TextField("Title", text: $text, onCommit: save)
                        TextField("Notes", text: $notes, onCommit: save)
                    }
                }
                Section {
                    ForEach(self.getItems(), id: \.self) { item in
                        Cell(item: item)
                    }
                    // TODO
                    //.onDelete(perform: )
                    //.onMove(perform: )
                }
            }
            .navigationBarTitle("Shopping List")
            .navigationBarItems(leading: EditButton(),
                                trailing: AddButton(isAdding,
                                                    action: toggleAdding))
        }
    }
    
    func getItems() -> [[String:String]] {
        // TODO: We want to use a function here to isolate schema knowledge inside replicant code.
        // Thus need to implement return values from Replicant.
        let req = ["key": "items"]
        let data = try! replicant.dispatch("get", data: JSONEncoder().encode(req))
        let resp = try! JSONSerialization.jsonObject(with: data)
        print(resp)
        return (resp as! [String:Any])["data"] as? [[String:String]] ?? []
    }
}

// MARK: Preview

#if DEBUG
struct ContentView_Previews: PreviewProvider {
    static var previews: some View {
        ContentView(replicant: nil)
    }
}
#endif

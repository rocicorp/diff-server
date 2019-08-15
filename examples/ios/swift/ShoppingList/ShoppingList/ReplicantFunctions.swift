//
//  ReplicantFunctions.swift
//  ShoppingList
//
//  Created by Aaron Boodman on 8/14/19.
//  Copyright © 2019 Eric Lewis, Inc. All rights reserved.
//

import Foundation

let replicantFunctions = #"""
function addItem(id, text, note) {
    var items = getItems();
    items.push({
        id: id,
        text: text,
        note: note,
    });
    db.put('items', items);
}

function getItems() {
    return db.get('items') || [];
}
"""#

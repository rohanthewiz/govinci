import XCTest

// Simulator pass for the todo app (examples/todoapp) — the SwiftUI analog of
// examples/todoapp/app_test.go. Requires the todo app in the framework:
//
//   ios/build.sh ./examples/todoapp
//
// and runs alone, since GovinciUITests drives the mobileapp demo bound by the
// default build:
//
//   xcodebuild test ... -only-testing:GovinciUITests/TodoAppUITests
//
// Beyond the CRUD flow it pins the regression the data-layer test cannot see:
// Go clearing the draft after Add must reach the *focused* TextField, whose
// local buffer otherwise swallows upstream writes (the echo/rewrite split in
// Renderer.swift's GovinciTextField).
final class TodoAppUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    func testAddClearsFocusedInput() throws {
        let app = XCUIApplication()
        app.launch()

        XCTAssertTrue(app.staticTexts["Todos"].waitForExistence(timeout: 10),
                      "initial tree did not render")

        let field = app.textFields["What needs doing?"]
        XCTAssertTrue(field.waitForExistence(timeout: 5))
        field.tap()
        field.typeText("Buy milk")

        // The field keeps focus across the button tap — exactly the state in
        // which the clear-on-submit rewrite used to be dropped.
        app.buttons["Add"].tap()
        XCTAssertTrue(app.staticTexts["1 item left"].waitForExistence(timeout: 5),
                      "add did not round-trip")
        XCTAssertTrue(app.staticTexts["Buy milk"].waitForExistence(timeout: 5),
                      "new row did not render")

        // An empty TextField reports its prompt as its value; both spellings
        // of "empty" are accepted, anything else is the stale draft.
        let after = (field.value as? String) ?? ""
        XCTAssertTrue(after.isEmpty || after == "What needs doing?",
                      "input not cleared after add, still shows: \(after)")

        // Consecutive adds must work without manual erasing — the point of
        // clearing the draft in the first place. (Re-tap: the assertion above
        // doesn't depend on focus surviving the Add tap, so neither should
        // this typing.)
        field.tap()
        field.typeText("Walk dog")
        app.buttons["Add"].tap()
        XCTAssertTrue(app.staticTexts["2 items left"].waitForExistence(timeout: 5),
                      "second add did not round-trip")
        XCTAssertTrue(app.staticTexts["Walk dog"].waitForExistence(timeout: 5))
    }
}

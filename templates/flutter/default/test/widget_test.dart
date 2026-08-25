import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:sandbox_flutter/main.dart';

void main() {
  testWidgets('counter increments', (WidgetTester tester) async {
    await tester.pumpWidget(const CounterApp());

    expect(find.byKey(const Key('counter')), findsOneWidget);
    expect(tester.widget<Text>(find.byKey(const Key('counter'))).data, '0');

    await tester.tap(find.byKey(const Key('increment')));
    await tester.pump();

    expect(tester.widget<Text>(find.byKey(const Key('counter'))).data, '1');
  });
}

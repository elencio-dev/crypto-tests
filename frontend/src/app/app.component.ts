import { Component } from '@angular/core';
import { StepTraceComponent } from './step-trace/step-trace.component';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [StepTraceComponent],
  templateUrl: './app.component.html',
  styleUrl: './app.component.css'
})
export class AppComponent {
  title = 'Cryptography Visual Lab';
}
